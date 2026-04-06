import 'package:flutter_test/flutter_test.dart';

import 'package:app/main.dart';

void main() {
  testWidgets('app shows the core object overview', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const IdletimeApp());

    expect(find.text('idletime'), findsOneWidget);
    expect(find.text('Core objects ready'), findsOneWidget);
    expect(find.text('Sport'), findsOneWidget);
    expect(find.text('Climbing'), findsOneWidget);
  });
}
