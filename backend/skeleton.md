
# P2P FileShare System

## Server/Client P2P Network Subsystem
- **Initialize Server Listener**, creating two streams:
  - `file_stream`: 
    - Protocol is established here.
    - Frames are sent over this stream.
  - `identity_stream`: 
    - Broadcasts identification information (e.g., **NAME, MAC, IP, TYPE**).
  
- **Launch mDNS service for both SERVER & CLIENT**:
  - **SERVER INSTANCE**:
    - Launch `identity` service protocol to exchange metadata.
    - Upon connection verification:
      - Launch `file_stream` service.
  - **CLIENT INSTANCE**:
    - Upon peer discovery:
      - Exchange metadata on `identity` and store in local `peerstore`.
    - When connecting to a peer:
      - Start writing to `file_stream`.

---

## Connection Verification & Authentication Subsystem
_(Details not specified)_

---

## Stateful Buffer Frame Multi-Modal Communications Subsystem
- **Create a `file_stream` channel.**
- **Ensure single-thread access at any given moment**:
  - **Thread X writes to `file_stream`** → **LOCK MUTEX**.
  - **Thread X finishes writing** → **UNLOCK MUTEX**.
- **Multiple writers** are initiated:
  - Each writer runs in its **own thread** via `go`.
  - Writers represent **different data sources** (e.g., Wi-Fi, USB, Bluetooth).
- **On completion**, send **COMPLETE SIGNAL** to the **Interrupt Handler Subsystem (IHS)**.

---

## File Reader/Writer Subsystem
- **Read from disk**.
- **Wrap data buffer** in a **stateful buffer frame**.
- **Serialize frame** using **FlatBuffers**.
- **Writer acquires frame**.
- **Read from `file_stream` channel**:
  - Only access: 
    - `OFFSET`
    - `DATA`
  - **Write DATA at OFFSET**.
- **If ERROR occurs**:
  - Send the frame to the **Interrupt Handler Subsystem (IHS)**.
  - **Pause process** (do not close) until IHS resolution.

---

## Interrupt Handler Subsystem (IHS)
- **Acquire frame with a signal**, which can be:
  - **ERROR**:
    - **Save FRAME + META** in the database.
    - Re-check connection every **X** seconds.
    - Upon successful connection → **Continue File Writer Subsystem (FWS)**.
  - **COMPLETE**:
    - Compute **HASH of the file**.
    - Compare with the **sent hash**.
  - **USER_STOP**:
    - **Save FRAME + META** in the database.
    - **Exit process**.
- **Process runs continuously**, waiting for signals.

---

## Restore / Save Subsystem
- Use **SQLite Database** to periodically store file states.
- **Before any operation**, check file state in the DB:
  - **If state is found**:
    - **Send RELOAD request** with state data to the server.
  - **If state is not found**:
    - **Initialize a new record** for the file.
- **Upon receiving a COMPLETE SIGNAL**:
  - **Delete records associated** with the operation.

