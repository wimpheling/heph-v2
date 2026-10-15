fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("cargo:rerun-if-changed=../../proto");
    connectrpc_build::Config::new()
        .files(&["../../proto/heph/system/v1/system.proto"])
        .includes(&["../../proto"])
        .include_file("_connectrpc.rs")
        .compile()?;
    Ok(())
}
