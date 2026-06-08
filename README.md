# Wayer - PC-End Server

Python-based server for transferring files between your PC and Android phone through hotspot connection.

**🔗 [← Back to Wayer](https://github.com/ojilon/Wayer/blob/main/README.md)** | **📖 [Detailed Setup Guide →](https://github.com/ojilon/WayerPC/blob/main/README_PC_END.md)**

## ⚡ Quick Start

```bash
# Install dependencies
pip install -r requirements.txt

# Navigate to python folder
cd python

# Start the server
python server/app3.py
```

You should see:
```
Listening on <HOST>:<PORT>
```

## 📂 Project Structure

```
WayerPC/
├── python/
│   ├── server/              # Main server code
│   │   ├── app3.py          # Entry point
│   │   ├── socket_server3.py
│   │   ├── Locate.py
│   │   ├── transfer.py
│   │   └── config.py
│   └── Filesmanager/        # File utilities
│
├── c/                       # C DLL source for file search
└── CMakeLists.txt
```

## 🎯 Server Operations

### `/ask` - File Download
Phone requests a file from the PC
```
/ask <filename>
```

### `/upload` - File Upload
Phone sends a file to the PC
```
/upload <filesize> <filename>
```

Files upload to the `received/` folder automatically.

## 📚 Full Documentation

For complete setup, configuration, and troubleshooting:
👉 **[README_PC_END.md](https://github.com/ojilon/WayerPC/blob/main/README_PC_END.md)**

## 🔗 Related Repository

- **Android Client**: [Wayer - android-end branch](https://github.com/ojilon/Wayer/tree/android-end)
