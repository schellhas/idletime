class Category {
  const Category({this.id, required this.name, this.multiplier = 1.0});

  static const tableName = 'categories';

  final int? id;
  final String name;
  final double multiplier;

  Category copyWith({int? id, String? name, double? multiplier}) {
    return Category(
      id: id ?? this.id,
      name: name ?? this.name,
      multiplier: multiplier ?? this.multiplier,
    );
  }

  Map<String, Object?> toMap() {
    return {'id': id, 'name': name, 'multiplier': multiplier};
  }

  factory Category.fromMap(Map<String, Object?> map) {
    return Category(
      id: map['id'] as int?,
      name: map['name'] as String,
      multiplier: (map['multiplier'] as num?)?.toDouble() ?? 1.0,
    );
  }
}
