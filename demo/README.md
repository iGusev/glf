# GLF Demo Files

This directory contains everything needed to create VHS demo recordings for the GLF project.

## Structure

```
demo/
├── data/               # Generated fake project data
│   └── glf/           # Fake GLF cache with projects, index, history
├── home/               # Temporary HOME directory for demos (created by setup)
├── demo.tape           # Main interactive demo
├── demo-basic.tape     # JSON API examples
├── demo-history.tape   # History feature demo
├── demo-dot.tape       # Debug: glf . command
├── demo-glf.sh         # Wrapper script to run GLF with fake data
├── demo-setup.sh       # Setup script: generates data and git repo
├── demo-cleanup.sh     # Cleanup script: removes temporary files
└── generate-fake-data.go  # Generator for fake project database
```

## Quick Start

```bash
# 1. Setup demo environment (run once or after cleanup)
./demo/demo-setup.sh

# 2. Record a demo
vhs demo/demo.tape           # Main demo
vhs demo/demo-basic.tape     # JSON examples
vhs demo/demo-history.tape   # History demo
vhs demo/demo-dot.tape       # Debug glf . command

# 3. Cleanup when done
./demo/demo-cleanup.sh
```

## Demo Files

### demo.tape
Main demonstration showing:
- Interactive fuzzy search
- Navigation through results
- Auto-open with `-g` flag
- Opening current git project with `glf .`

### demo-basic.tape
JSON API examples for integration documentation

### demo-history.tape
History tracking and usage-based ranking demonstration

### demo-dot.tape
Debug version focusing only on the `glf .` command

## How It Works

1. **generate-fake-data.go**: Creates 37 diverse fake GitLab projects with realistic data
2. **demo-glf.sh**: Wrapper that runs GLF with:
   - Fake HOME directory (`demo/home`)
   - Fake config pointing to `gitlab.company.com`
   - Symlinked cache to `demo/data/glf`
3. **demo-setup.sh**:
   - Runs `generate-fake-data.go` to create cache
   - Creates `~/projects/backend-api` with git remote
4. **VHS tapes**: Record terminal sessions using the wrapper

## Notes

- The `demo/data/` directory is preserved between cleanups for faster re-recording
- The `demo/home/` directory is recreated on each wrapper run
- All demos use the wrapper script to avoid exposing real GitLab data
- The fake git repository at `~/projects/backend-api` is cleaned up after demos
