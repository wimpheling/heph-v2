fn main() {
    let _ = std::env::var("HEPH_LISTEN_ADDR");
    let _ = std::env::var_os("HEPH_LISTEN_ADDR");
    let _ = std::env::vars();
    let _ = std::env::vars_os();
}
