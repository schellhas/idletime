import 'package:sqflite/sqflite.dart';

import '../data/database/database_helper.dart';
import '../models/category.dart';

class CategoryRepository {
  CategoryRepository({DatabaseHelper? databaseHelper})
    : _databaseHelper = databaseHelper ?? DatabaseHelper.instance;

  final DatabaseHelper _databaseHelper;

  Future<List<Category>> getAllCategories() async {
    final db = await _databaseHelper.database;
    final maps = await db.query(Category.tableName, orderBy: 'name ASC');
    return maps.map(Category.fromMap).toList();
  }

  Future<Category> insertCategory(Category category) async {
    final db = await _databaseHelper.database;
    final id = await db.insert(
      Category.tableName,
      category.toMap()..remove('id'),
      conflictAlgorithm: ConflictAlgorithm.replace,
    );
    return category.copyWith(id: id);
  }

  Future<int> updateCategory(Category category) async {
    if (category.id == null) {
      throw ArgumentError('Category id cannot be null for updates.');
    }

    final db = await _databaseHelper.database;
    return db.update(
      Category.tableName,
      category.toMap()..remove('id'),
      where: 'id = ?',
      whereArgs: [category.id],
    );
  }

  Future<int> deleteCategory(int id) async {
    final db = await _databaseHelper.database;
    return db.delete(Category.tableName, where: 'id = ?', whereArgs: [id]);
  }
}
