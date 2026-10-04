use std::io::{self, Write};

use tracing_subscriber::{EnvFilter, fmt::MakeWriter};

/// Line-oriented stdout writer that flushes every write.
///
/// Render captures process stdout through a pipe. Rust block-buffers stdout
/// when it is not a terminal, so a normal tracing subscriber can sit silent
/// until the buffer fills. Flushing each event makes startup, request, and
/// research logs show up as they happen.
struct FlushStdout;

impl Write for FlushStdout {
    fn write(&mut self, buf: &[u8]) -> io::Result<usize> {
        let mut stdout = io::stdout().lock();
        stdout.write_all(buf)?;
        stdout.flush()?;
        Ok(buf.len())
    }

    fn flush(&mut self) -> io::Result<()> {
        io::stdout().flush()
    }
}

impl<'a> MakeWriter<'a> for FlushStdout {
    type Writer = FlushStdout;

    fn make_writer(&'a self) -> Self::Writer {
        FlushStdout
    }
}

pub fn init() -> anyhow::Result<()> {
    let filter = EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info"));
    tracing_subscriber::fmt()
        .with_env_filter(filter)
        .with_ansi(false)
        .with_target(true)
        .with_writer(FlushStdout)
        .try_init()
        .map_err(|error| anyhow::anyhow!("failed to initialize tracing: {error}"))?;
    tracing::info!("logging initialized on stdout");
    Ok(())
}
