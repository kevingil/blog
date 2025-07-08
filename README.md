# Blog Copilot

An agentic blog editor.


![frontend/public/Xnip2025-06-29_15-58-49.jpg](frontend/public/Xnip2025-07-08_00-16-13.png)


![frontend/public/Xnip2025-06-29_15-58-49.jpg](frontend/public/Xnip2025-06-29_15-58-49.jpg)


## Getting Started

### Frontend Setup

Navigate to the frontend directory and install dependencies:
```bash
cd frontend
bun install
```

Start the development server:
```bash
bun run dev
```

Build for production:
```bash
bun run build
```

### Backend Setup (Go)

The Makefile commands are for the backend only:

Run build make command with tests:
```bash
make all
```

Build the application:
```bash
make build
```

Run the application:
```bash
make run
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```
