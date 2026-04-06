# Technical Stack

This file captures the current technical direction for `idletime`.

## Stack

- **App framework:** Flutter
- **Language:** Dart
- **Database:** SQLite
- **Targets:**
  - Web version for **desktop** and **mobile** browsers
  - App for **iOS** and **Android**

## Architecture direction

The current direction is **local-first**:

- the app is built in **Flutter/Dart**
- app logic, services, and repositories live inside the app
- data is stored locally on the device in SQLite
- there are **no user accounts** and **no authentication** for now
- users can manually back up their data and move it to another device

## Backup / transfer

To switch devices, the user should be able to:

- export the local database file
- store it wherever they want
- import that file on another device

## Notes

- One shared Flutter codebase is the current goal for web, iOS, and Android.
- A separate backend is not planned for the first version.
- This document will be expanded step by step.
