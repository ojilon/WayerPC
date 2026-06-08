# Wayer - PC End

A Python-based server for transferring files between your PC and Android phone through hotspot connection.

## Quick Start

### Prerequisites

- Python 3.x installed on your PC.
- MinGW64 or GCC (For C)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/ojilon/WayerPC.git
   cd WayerPC
   ```

### Running the Server

Navigate to the python folder and start the server:

```bash
cd python
python server/app3.py
```

The server will start listening on the configured host and port. You should see:
```
Listening on <HOST>:<PORT>
```

## Project Structure

```
WayerPC/
├── python/
│   ├── server/
│   │   ├── app3.py              # Main entry point - start the server here
│   │   ├── socket_server3.py    # Socket server handling client connections
│   │   ├── Locate.py            # File location utilities
│   │   ├── transfer.py          # File transfer logic
│   │   └── config.py            # Configuration settings
│   └── Filesmanager/
│       └── FindRoot.py          # Root directory finder
├── c/                           # C code for file search functionality
├── build/                       # Build output directory
└── README.md
```

## How It Works

**The server runs two main operations:**

### 1. **File Download (/ask command)**
- The Android phone requests a file from the PC using the `/ask` command
- The server searches for the file in the project and `shared/` folder
- If found, the file is streamed to the phone
- Uses a C DLL (`libfilesearch.dll`) for fast file searching

### 2. **File Upload (/upload command)**
- The Android phone sends a file to the PC using `/upload` command
- Files are saved to the `received/` folder
- The server streams the file data and saves it to disk

## Key Components

| File | Purpose |
|------|----------|
| `app3.py` | Entry point - starts the server |
| `socket_server3.py` | Core server logic handling `/ask` and `/upload` commands |
| `Locate.py` | Utilities for finding files and folders, locating DLL |
| `transfer.py` | File streaming and transfer functions |
| `config.py` | Server configuration (HOST, PORT, etc.) |

## Folders

- **`shared/`** - Contains files available for download by the phone (auto-created if missing)
- **`received/`** - Stores files uploaded from the phone (auto-created if missing)

## Troubleshooting

- **DLL not found error**: Make sure `libfilesearch.dll` is in the `c/` folder
- **Socket address in use**: Port may be occupied; check `config.py` and change PORT
- **Folder creation failed**: Ensure the application has write permissions in the project directory
- **Connection issues**: Verify the PC and phone are connected to the same hotspot

## Notes

- The server creates necessary folders (`shared/`, `received/`) automatically if they don't exist
- All file operations are logged to console
- The C DLL handles efficient file system searching
