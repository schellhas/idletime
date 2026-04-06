import 'package:flutter/material.dart';

import 'models/activity.dart';
import 'models/category.dart';
import 'repositories/activity_repository.dart';
import 'repositories/category_repository.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
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

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  final _categoryRepository = CategoryRepository();
  final _activityRepository = ActivityRepository();

  late Future<_HomeData> _homeDataFuture;

  @override
  void initState() {
    super.initState();
    _homeDataFuture = _loadHomeData();
  }

  Future<_HomeData> _loadHomeData() async {
    await _seedDemoDataIfEmpty();

    final categories = await _categoryRepository.getAllCategories();
    final activities = await _activityRepository.getAllActivities();

    return _HomeData(categories: categories, activities: activities);
  }

  Future<void> _seedDemoDataIfEmpty() async {
    final existingCategories = await _categoryRepository.getAllCategories();
    if (existingCategories.isNotEmpty) {
      return;
    }

    final sport = await _categoryRepository.insertCategory(
      const Category(name: 'Sport', multiplier: 2.0),
    );

    if (sport.id == null) {
      return;
    }

    await _activityRepository.insertActivity(
      Activity(
        categoryId: sport.id!,
        name: 'Climbing',
        multiplier: 1.5,
        minimumMinutes: 60,
        trackedMinutes: 120,
      ),
    );
  }

  Future<void> _refresh() async {
    setState(() {
      _homeDataFuture = _loadHomeData();
    });

    await _homeDataFuture;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('idletime'),
        actions: [
          IconButton(
            onPressed: _refresh,
            tooltip: 'Refresh',
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: FutureBuilder<_HomeData>(
        future: _homeDataFuture,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }

          if (snapshot.hasError) {
            return Center(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text('Failed to load local data: ${snapshot.error}'),
              ),
            );
          }

          final homeData = snapshot.data!;
          final categoryNamesById = {
            for (final category in homeData.categories)
              if (category.id != null) category.id!: category.name,
          };

          return ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Text(
                'Saved in SQLite',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 8),
              Text(
                '${homeData.categories.length} categories · '
                '${homeData.activities.length} activities',
              ),
              const SizedBox(height: 16),
              Text(
                'Categories',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              ...homeData.categories.map(
                (category) => Card(
                  child: ListTile(
                    title: Text(category.name),
                    subtitle: Text(
                      'Category multiplier: ${category.multiplier}',
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Text(
                'Activities',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              ...homeData.activities.map(
                (activity) => Card(
                  child: ListTile(
                    title: Text(activity.name),
                    subtitle: Text(
                      'Category: '
                      '${categoryNamesById[activity.categoryId] ?? 'Unknown'}\n'
                      'Activity multiplier: ${activity.multiplier}\n'
                      'Minimum time: ${activity.minimumMinutes} min\n'
                      'Tracked time: ${activity.trackedMinutes} min',
                    ),
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _HomeData {
  const _HomeData({required this.categories, required this.activities});

  final List<Category> categories;
  final List<Activity> activities;
}
