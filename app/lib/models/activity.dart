class Activity {
  const Activity({
    this.id,
    required this.categoryId,
    required this.name,
    this.multiplier = 1.0,
    this.minimumMinutes = 0,
    this.trackedMinutes = 0,
  });

  static const tableName = 'activities';

  final int? id;
  final int categoryId;
  final String name;
  final double multiplier;
  final int minimumMinutes;
  final int trackedMinutes;

  Duration get minimumDuration => Duration(minutes: minimumMinutes);
  Duration get trackedDuration => Duration(minutes: trackedMinutes);

  Activity copyWith({
    int? id,
    int? categoryId,
    String? name,
    double? multiplier,
    int? minimumMinutes,
    int? trackedMinutes,
  }) {
    return Activity(
      id: id ?? this.id,
      categoryId: categoryId ?? this.categoryId,
      name: name ?? this.name,
      multiplier: multiplier ?? this.multiplier,
      minimumMinutes: minimumMinutes ?? this.minimumMinutes,
      trackedMinutes: trackedMinutes ?? this.trackedMinutes,
    );
  }

  Map<String, Object?> toMap() {
    return {
      'id': id,
      'category_id': categoryId,
      'name': name,
      'multiplier': multiplier,
      'minimum_minutes': minimumMinutes,
      'tracked_minutes': trackedMinutes,
    };
  }

  factory Activity.fromMap(Map<String, Object?> map) {
    return Activity(
      id: map['id'] as int?,
      categoryId: map['category_id'] as int,
      name: map['name'] as String,
      multiplier: (map['multiplier'] as num?)?.toDouble() ?? 1.0,
      minimumMinutes: map['minimum_minutes'] as int? ?? 0,
      trackedMinutes: map['tracked_minutes'] as int? ?? 0,
    );
  }
}
