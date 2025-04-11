use extism_pdk::{plugin_fn, FnResult};
use indexmap::IndexMap;
use serde::Deserialize;
use serde_with::{base64::Base64, serde_as};
use wac_graph::types::{BorrowedPackageKey, Package};
use wac_graph::{CompositionGraph, EncodeOptions};
use wac_parser::Document;

#[derive(Deserialize)]
struct Context {
    script: String,
    dependencies: Vec<Dependency>,
}

#[serde_as]
#[derive(Deserialize)]
struct Dependency {
    name: String,
    #[serde_as(as = "Base64")]
    component: Vec<u8>,
}

#[plugin_fn]
pub fn compose(input: Vec<u8>) -> FnResult<Vec<u8>> {
    let input: Context = serde_json::from_slice(&input)?;
    let document = Document::parse(input.script.as_str())?;
    let mut dependencies = IndexMap::new();
    for dep in input.dependencies.iter() {
        dependencies.insert(
            BorrowedPackageKey::from_name_and_version(dep.name.as_str(), None),
            dep.component.to_vec(),
        );
    }
    let resolution = document.resolve(dependencies)?;
    let bytes = resolution.encode(EncodeOptions::default())?;

    Ok(bytes)
}

#[plugin_fn]
pub fn plug(input: Vec<u8>) -> FnResult<Vec<u8>> {
    let input: Context = serde_json::from_slice(&input)?;

    let mut graph = CompositionGraph::new();

    let socket = input.dependencies.first().unwrap();
    let socket = Package::from_bytes(
        &socket.name,
        None,
        socket.component.clone(),
        graph.types_mut(),
    )?;
    let socket = graph.register_package(socket)?;

    let mut plugs = Vec::new();
    for plug in input.dependencies.iter().skip(1) {
        let plug =
            Package::from_bytes(&plug.name, None, plug.component.clone(), graph.types_mut())?;
        let plug = graph.register_package(plug)?;
        plugs.push(plug);
    }

    wac_graph::plug(&mut graph, plugs, socket)?;
    let bytes = graph.encode(EncodeOptions::default())?;

    Ok(bytes)
}
