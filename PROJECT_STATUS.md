# Project Status

_Last updated: 2026-04-06_

## Current direction

`idletime` is currently being built as a **local-first Flutter app**.

### Core decisions
- **Framework:** Flutter
- **Language:** Dart
- **Database:** SQLite
- **Targets:** web, iOS, Android
- **Architecture:** local-first
- **Users/auth:** not planned for v1
- **Data transfer:** manual export/import later

## Current product concept

The app helps decide what to do with a small amount of available time.

The user:
- defines categories and activities
- assigns multipliers to express relative priorities
- logs time spent on activities
- gets recommendations based on what is currently most behind

## What is already done

### Documentation
- `README.md` contains the product explanation and example usage
- `TECH_STACK.md` documents the technical choices
- `DOMAIN_MODEL.md` documents the core objects and storage direction

### Project setup
- Flutter SDK installed locally
- Flutter app created in `app/`
- root `.gitignore` added
- Flutter `.gitignore` already present in `app/`

### Implementation
- core models created:
  - `Category`
  - `Activity`
- SQLite layer created:
  - `database_helper.dart`
  - `CategoryRepository`
  - `ActivityRepository`
- app loads demo data from SQLite on startup

## Verified status

The following checks succeeded:

```bash
cd /home/a/repos/idletime/app
dart format lib test
flutter analyze
flutter test
flutter build web
```

Verified results:
- `flutter analyze` → no issues found
- `flutter test` → all tests passed
- `flutter build web` → build succeeded

## Current preview

A local preview has been started successfully in the browser before.

Typical run command:

```bash
source ~/.bashrc
cd /home/a/repos/idletime/app
flutter run -d web-server --web-port 8081
```

## Current TODOs

### Immediate next steps
- [ ] build UI for creating categories
- [ ] build UI for creating activities
- [ ] connect the forms to the repositories
- [ ] show real saved categories and activities cleanly in the app

### After that
- [ ] add manual time logging
- [ ] add timer start / stop flow
- [ ] implement recommendation logic
- [ ] add graphs / progress view
- [ ] add export / import for local backup transfer

### Later
- [ ] improve mobile layout
- [ ] evaluate iOS and Android packaging
- [ ] decide whether any sync/backend is needed at all

## Important files

- `README.md`
- `TECH_STACK.md`
- `DOMAIN_MODEL.md`
- `PROJECT_STATUS.md`
- `app/lib/main.dart`
- `app/lib/models/category.dart`
- `app/lib/models/activity.dart`
- `app/lib/data/database/database_helper.dart`
- `app/lib/repositories/category_repository.dart`
- `app/lib/repositories/activity_repository.dart`
