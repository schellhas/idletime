import 'package:sqflite/sqflite.dart';

import '../data/database/database_helper.dart';
import '../models/activity.dart';

class ActivityRepository {
  ActivityRepository({DatabaseHelper? databaseHelper})
    : _databaseHelper = databaseHelper ?? DatabaseHelper.instance;

  final DatabaseHelper _databaseHelper;

  Future<List<Activity>> getAllActivities() async {
    final db = await _databaseHelper.database;
    final maps = await db.query(Activity.tableName, orderBy: 'name ASC');
    return maps.map(Activity.fromMap).toList();
  }

  Future<List<Activity>> getActivitiesForCategory(int categoryId) async {
    final db = await _databaseHelper.database;
    final maps = await db.query(
      Activity.tableName,
      where: 'category_id = ?',
      whereArgs: [categoryId],
      orderBy: 'name ASC',
    );
    return maps.map(Activity.fromMap).toList();
  }

  Future<Activity> insertActivity(Activity activity) async {
    final db = await _databaseHelper.database;
    final id = await db.insert(
      Activity.tableName,
      activity.toMap()..remove('id'),
      conflictAlgorithm: ConflictAlgorithm.replace,
    );
    return activity.copyWith(id: id);
  }

  Future<int> updateActivity(Activity activity) async {
    if (activity.id == null) {
      throw ArgumentError('Activity id cannot be null for updates.');
    }

    final db = await _databaseHelper.database;
    return db.update(
      Activity.tableName,
      activity.toMap()..remove('id'),
      where: 'id = ?',
      whereArgs: [activity.id],
    );
  }

  Future<int> deleteActivity(int id) async {
    final db = await _databaseHelper.database;
    return db.delete(Activity.tableName, where: 'id = ?', whereArgs: [id]);
  }
}
