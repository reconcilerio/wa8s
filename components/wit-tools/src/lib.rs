use extism_pdk::{plugin_fn, FnResult};
use wat::Detect;
use wit_component::{DecodedWasm, WitPrinter};

#[plugin_fn]
pub fn extract(input: Vec<u8>) -> FnResult<String> {
    let wit = match Detect::from_bytes(&input) {
        Detect::WasmBinary | Detect::WasmText => {
            // Use `wat` to possible translate the text format, and then
            // afterwards use either `decode` or `metadata::decode` depending on
            // if the input is a component or a core wasm module.
            let input = wat::parse_bytes(&input)?;
            if wasmparser::Parser::is_component(&input) {
                wit_component::decode(&input)
            } else {
                let (wasm, bindgen) = wit_component::metadata::decode(&input)?;
                if wasm.is_none() {
                    panic!(
                        "input is a core wasm module with no `component-type*` \
                         custom sections meaning that there is not WIT information; \
                         is the information not embedded or is this supposed \
                         to be a component?"
                    )
                }
                Ok(DecodedWasm::Component(bindgen.resolve, bindgen.world))
            }
        }
        Detect::Unknown => {
            panic!("unknown blob format");
        }
    }?;

    let mut printer = WitPrinter::default();
    printer.print(wit.resolve(), wit.package(), &vec![])?;

    Ok(printer.output.to_string())
}
