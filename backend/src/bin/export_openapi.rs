use std::path::PathBuf;

use clap::Parser;

#[derive(Debug, Parser)]
struct Args {
    #[arg(long, default_value = "../openapi/openapi.json")]
    output: PathBuf,
}

fn main() -> anyhow::Result<()> {
    let args = Args::parse();
    let json = blog_backend::openapi::canonical_json()?;
    if let Some(parent) = args.output.parent() {
        std::fs::create_dir_all(parent)?;
    }
    std::fs::write(args.output, json)?;
    Ok(())
}
