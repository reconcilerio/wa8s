use anyhow::{Context, Result};
use config::{create_config, Config};
use std::collections::HashMap;
use wasm_metadata::Producers;
use wit_component::{metadata, ComponentEncoder, DecodedWasm, StringEncoding};

pub mod config;
mod walrus_ops;

const ADAPTER: &[u8] = include_bytes!("../lib/adapter.wasm");
const WIT_METADATA: &[u8] = include_bytes!("../lib/package.wasm");

pub fn create_component(values: Vec<(String, String)>) -> Result<Vec<u8>> {
    let mut config = walrus::ModuleConfig::new();
    // TODO reconsider?
    config.generate_name_section(false);
    let mut module = config.parse(ADAPTER)?;
    module.name = Some("static-config".into());

    let mut properties = HashMap::new();
    for (key, value) in values {
        properties.insert(key, value);
    }

    let config = &Config {
        overrides: properties.into_iter().collect(),
    };
    create_config(&mut module, config)?;

    let component_type_section_id = module
        .customs
        .iter()
        .find_map(|(id, section)| {
            let name = section.name();
            if name.starts_with("component-type:")
                && section.as_any().is::<walrus::RawCustomSection>()
            {
                Some(id)
            } else {
                None
            }
        })
        .context("Unable to find component type section")?;

    // decode the component custom section to strip out the unused world exports
    // before reencoding.
    let mut component_section = *module
        .customs
        .delete(component_type_section_id)
        .context("Unable to find component section")?
        .into_any()
        .downcast::<walrus::RawCustomSection>()
        .unwrap();

    let (resolve, pkg_id) = match wit_component::decode(WIT_METADATA)? {
        DecodedWasm::WitPackage(resolve, pkg_id) => (resolve, pkg_id),
        DecodedWasm::Component(..) => {
            anyhow::bail!("expected a WIT package, found a component")
        }
    };

    let world = resolve.select_world(pkg_id, Some("adapter"))?;

    let mut producers = Producers::default();
    producers.add("processed-by", "static-config", env!("CARGO_PKG_VERSION"));

    component_section.data =
        metadata::encode(&resolve, world, StringEncoding::UTF8, Some(&producers))?;

    module.customs.add(component_section);

    let bytes = module.emit_wasm();

    // now adapt the virtualized component
    let mut encoder = ComponentEncoder::default().validate(true).module(&bytes)?;
    let encoded = encoder.encode()?;

    Ok(encoded)
}
