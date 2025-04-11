use extism_pdk::*;
use static_config::create_component;

#[plugin_fn]
pub fn build_component(Json(values): Json<Vec<(String, String)>>) -> FnResult<Vec<u8>> {
    Ok(create_component(values)?)
}
