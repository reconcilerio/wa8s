#![no_main]

use crate::exports::reconcilerio::config::factory::{Error, Guest};
use static_config::create_component;

pub(crate) struct Factory;

impl Guest for Factory {
    fn build_component(values: Vec<(String, String)>) -> Result<Vec<u8>, Error> {
        let output = create_component(values).map_err(|e| Error::from(e.to_string()))?;
        Ok(output)
    }
}

wit_bindgen::generate!({
    path: "../wit",
    world: "config-factory",
    generate_all
});

export!(Factory);
