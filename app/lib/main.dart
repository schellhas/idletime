import 'package:flutter/material.dart';

import 'models/activity.dart';
import 'models/category.dart';

void main() {
  runApp(const IdletimeApp());
}

class IdletimeApp extends StatelessWidget {
  const IdletimeApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'idletime',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.indigo),
      ),
      home: const HomeScreen(),
    );
  }
}

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    const sport = Category(id: 1, name: 'Sport', multiplier: 2.0);
    const climbing = Activity(
      id: 1,
      categoryId: 1,
      name: 'Climbing',
      multiplier: 1.5,
      minimumMinutes: 60,
      trackedMinutes: 120,
    );

    return Scaffold(
      appBar: AppBar(title: const Text('idletime')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Core objects ready',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 16),
            Card(
              child: ListTile(
                title: Text(sport.name),
                subtitle: Text('Category multiplier: ${sport.multiplier}'),
              ),
            ),
            Card(
              child: ListTile(
                title: Text(climbing.name),
                subtitle: Text(
                  'Activity multiplier: ${climbing.multiplier}\n'
                  'Minimum time: ${climbing.minimumMinutes} min\n'
                  'Tracked time: ${climbing.trackedMinutes} min',
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
