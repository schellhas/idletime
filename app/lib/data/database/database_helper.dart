import 'package:flutter/foundation.dart'
    show TargetPlatform, defaultTargetPlatform, kIsWeb;
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';
import 'package:sqflite_common_ffi_web/sqflite_ffi_web.dart';

import '../../models/activity.dart';
import '../../models/category.dart';

class DatabaseHelper {
  DatabaseHelper._();

  static final DatabaseHelper instance = DatabaseHelper._();

  static const _databaseName = 'idletime.db';
  static const _databaseVersion = 1;

  Database? _database;

  Future<Database> get database async {
    if (_database != null) {
      return _database!;
    }

    await _configureDatabaseFactory();
    final databasePath = await _resolveDatabasePath();

    _database = await openDatabase(
      databasePath,
      version: _databaseVersion,
      onConfigure: (db) async {
        await db.execute('PRAGMA foreign_keys = ON');
      },
      onCreate: _onCreate,
    );

    return _database!;
  }

  Future<void> _configureDatabaseFactory() async {
    if (kIsWeb) {
      databaseFactory = databaseFactoryFfiWeb;
      return;
    }

    switch (defaultTargetPlatform) {
      case TargetPlatform.linux:
      case TargetPlatform.windows:
        sqfliteFfiInit();
        databaseFactory = databaseFactoryFfi;
        break;
      case TargetPlatform.android:
      case TargetPlatform.iOS:
      case TargetPlatform.macOS:
      case TargetPlatform.fuchsia:
        break;
    }
  }

  Future<String> _resolveDatabasePath() async {
    if (kIsWeb) {
      return _databaseName;
    }

    switch (defaultTargetPlatform) {
      case TargetPlatform.linux:
      case TargetPlatform.windows:
        final appSupportDirectory = await getApplicationSupportDirectory();
        return path.join(appSupportDirectory.path, _databaseName);
      case TargetPlatform.android:
      case TargetPlatform.iOS:
      case TargetPlatform.macOS:
      case TargetPlatform.fuchsia:
        final databasesPath = await getDatabasesPath();
        return path.join(databasesPath, _databaseName);
    }
  }

  Future<void> _onCreate(Database db, int version) async {
    await db.execute('''
      CREATE TABLE ${Category.tableName} (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        multiplier REAL NOT NULL DEFAULT 1.0
      )
    ''');

    await db.execute('''
      CREATE TABLE ${Activity.tableName} (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        category_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        multiplier REAL NOT NULL DEFAULT 1.0,
        minimum_minutes INTEGER NOT NULL DEFAULT 0,
        tracked_minutes INTEGER NOT NULL DEFAULT 0,
        FOREIGN KEY (category_id) REFERENCES ${Category.tableName}(id) ON DELETE CASCADE
      )
    ''');
  }
}
