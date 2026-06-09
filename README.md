# 💻 Wayer - PC Server

Python-based server for transferring files between your PC and Android phone through hotspot connection.

**Wayer** is a two-component project:
- 💻 **PC Server** - File server for handling requests (This Repository)
- 📱 **Android Client** - Connect and transfer files from your phone

---

## 🚀 Quick Start

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

---

## 📋 System Requirements

- Python 3.x
- MinGW64 or GCC (for C DLL compilation)
- CMake 3.10+ (for building C DLL)

---

## 🔗 Related Component

### 📱 **Android Client** (Separate Repository)

Install and run the Android app on your phone to connect and transfer files.

**📖 [→ Full Android Setup Guide](https://github.com/ojilon/Wayer/blob/main/README_ANDROID_END.md)**  
**🔗 [→ Go to Wayer Repository](https://github.com/ojilon/Wayer)**

---

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
│       └── FindRoot.py
├── c/                       # C DLL source for file search
├── build/                   # Build output directory
├── CMakeLists.txt
├── .gitignore
├── LICENSE
├── README.md                # Quick overview
└── README_PC_END.md         # Detailed guide
```

---

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

Files automatically upload to the `received/` folder.

---

## 📚 Full Documentation

For complete setup, configuration, compilation, and troubleshooting:

👉 **[README_PC_END.md](https://github.com/ojilon/WayerPC/blob/main/README_PC_END.md)**

---

## 🏗️ Project Architecture

```
Wayer (Complete File Transfer System)
├── ojilon/WayerPC (This Repo)
│   ├── 💻 Python Server + C DLL
│   ├── Languages: Python, C, C++, CMake
│   ├── Purpose: File server for handling requests
│   └── Role: Backend processing & file search
│
└── ojilon/Wayer (Separate Repo)
    ├── 📱 Android Application
    ├── Language: Javas
    ├── Build: Gradle
    └── Purpose: Client app for file browsing & transfer
```

---

## 🔄 How It Works

### System Flow

1. **Start this server on your PC**
   ```bash
   python server/app3.py
   ```

2. **Install Wayer app on your Android phone**
   - Use the Android repository to build the APK

3. **Connect phone to PC hotspot**
   - Both devices need to be on the same network

4. **Use the app to transfer files**
   - List files: `ls`
   - Navigate: `cd <path>`
   - Download: `/ask <filename>`
   - Upload: `/upload <filepath>`

---

## 📁 Folders

- **`shared/`** - Contains files available for download by the phone (auto-created if missing)
- **`received/`** - Stores files uploaded from the phone (auto-created if missing)
- **`python/`** - Main server code and utilities
- **`c/`** - C files
- **`build/`** - CMake build output directory + DLL file

---

## 🎮 Server Features

✅ Handles `/ask` file download requests  
✅ Handles `/upload` file upload requests  
✅ Fast file searching with C DLL  
✅ Automatic folder creation  
✅ Connection logging and status reporting  
✅ Optimized for hotspot connections  

---

## 🛠️ Building the C DLL

The server uses a C DLL (`libfilesearch.dll`) for efficient file system searching.

To compile:
```bash
# Install dependencies
# - CMake 3.10+
# - MinGW64 or GCC

# Create and enter build directory
mkdir build
cd build

# Generate build files
cmake ..

# Build
cmake --build .
```

The compiled DLL will be in `build/c/`.

---

## 📊 Key Components

| File | Purpose |
|------|---------|
| `app3.py` | Entry point - starts the server |
| `socket_server3.py` | Core server logic handling `/ask` and `/upload` commands |
| `Locate.py` | Utilities for finding files and folders, locating DLL |
| `transfer.py` | File streaming and transfer functions |
| `config.py` | Server configuration (HOST, PORT, etc.) |

---

## 🔧 Configuration

Edit `config.py` to customize server settings:
- HOST: Server listening address
- PORT: Server listening port
- Other connection parameters

---

## 🚨 Troubleshooting

| Issue | Solution |
|-------|----------|
| **DLL not found error** | Make sure `libfilesearch.dll` is in the `c/` folder after compilation |
| **Socket address in use** | Port may be occupied; change PORT in `config.py` |
| **Folder creation failed** | Ensure the application has write permissions in the project directory |
| **Connection issues** | Verify the PC and phone are connected to the same hotspot |
| **File not found** | Check if the file exists in the project or shared folder |

---

## 📝 Notes

- The server creates necessary folders (`shared/`, `received/`) automatically if they don't exist
- All file operations are logged to console
- The C DLL handles efficient file system searching
- Both devices must be on the same network (hotspot or WiFi)
- Larger files may take longer depending on connection quality

---

## 🔗 Links

- **PC Server Repo**: https://github.com/ojilon/WayerPC
- **Android Client Repo**: https://github.com/ojilon/Wayer
- **Android Setup Guide**: [README_ANDROID_END.md](https://github.com/ojilon/Wayer/blob/main/README_ANDROID_END.md)
- **PC Server Full Guide**: [README_PC_END.md](https://github.com/ojilon/WayerPC/blob/main/README_PC_END.md)

---

## 📝 License

MIT License - See LICENSE file

---

**Get started**: [Full PC Server Guide →](https://github.com/ojilon/WayerPC/blob/main/README_PC_END.md) | [Android App Guide →](https://github.com/ojilon/Wayer/blob/main/README_ANDROID_END.md)
