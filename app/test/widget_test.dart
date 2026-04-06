import 'package:flutter_test/flutter_test.dart';

import 'package:app/models/activity.dart';
import 'package:app/models/category.dart';

void main() {
  test('category and activity map serialization works', () {
    const category = Category(id: 1, name: 'Sport', multiplier: 2.0);
    const activity = Activity(
      id: 2,
      categoryId: 1,
      name: 'Climbing',
      multiplier: 1.5,
      minimumMinutes: 60,
      trackedMinutes: 120,
    );

    expect(Category.fromMap(category.toMap()).name, 'Sport');
    expect(Activity.fromMap(activity.toMap()).name, 'Climbing');
    expect(Activity.fromMap(activity.toMap()).trackedMinutes, 120);
  });
}
