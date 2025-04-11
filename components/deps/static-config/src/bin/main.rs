use anyhow::{Error, Result};
use clap::Parser;
use static_config::create_component;
use std::{
    fs::File,
    io::{self, Read, Write},
    path::Path,
};

fn main() {
    if let Err(message) = Command::parse().exec() {
        eprintln!("{message}");
        std::process::exit(1);
    }
}

#[derive(Debug, Parser)]
#[command()]
/// Expose static configuration as a wasm component exporting the
/// wasi:config/store interface.
struct Command {
    /// Output to a file, '-' for stdout
    #[arg(short('o'), long("output"), name("wasm-file"))]
    output: String,
    /// Properties files, '-' for stdin
    #[arg(short('f'), long("file"), name("properties-file"))]
    paths: Option<Vec<String>>,
    /// Individual properties key=value
    #[arg(short('p'), long("property"), name("property"))]
    properties: Option<Vec<String>>,
}

impl Command {
    fn exec(&self) -> Result<()> {
        let mut properties: Vec<(String, String)> = vec![];

        for path in self.paths.clone().unwrap_or(vec![]) {
            let input = match path.as_str() {
                "-" => {
                    eprintln!("Reading properties from stdin (ctrl-c to exit)");
                    Box::new(io::stdin()) as Box<dyn Read>
                }
                _ => {
                    eprintln!("Reading properties file from {path}");
                    let path = Path::new(path.as_str());
                    Box::new(File::open(&path)?) as Box<dyn Read>
                }
            };
            properties.extend(java_properties::read(input)?);
        }

        for property in self.properties.clone().unwrap_or(vec![]) {
            let parts: Vec<&str> = property.splitn(2, "=").collect();
            if parts.len() != 2 {
                Result::Err(Error::msg("Property must take form key=value"))?;
            }
            properties.push((parts[0].to_string(), parts[1].to_string()));
        }

        let mut output = match self.output.as_str() {
            "-" => {
                eprintln!("Writing component to stdout");
                Box::new(io::stdout()) as Box<dyn Write>
            }
            _ => {
                let path = self.output.clone();
                eprintln!("Writing component to {path}");
                let path = Path::new(path.as_str());
                Box::new(File::create(&path)?) as Box<dyn Write>
            }
        };

        let component = create_component(properties)?;
        output.write_all(&component).map_err(|e| Error::from(e))
    }
}
