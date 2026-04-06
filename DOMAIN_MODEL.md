# Domain Model

This file describes the core objects of `idletime` and how they relate to each other.

## Core objects

At the moment, the model is intentionally small and consists of two main objects:

- **Category**
- **Activity**

## Category

A category groups activities together.

Examples:
- `sport`
- `self-care`
- `languages`
- `leisure`
- `productivity`

A category can also have its own multiplier so that some areas of life are prioritized more than others.

## Activity

An activity belongs to a category.

Examples:
- `swimming`
- `jogging`
- `gym`
- `reading`
- `journaling`
- `spanish`

Each activity can have:
- a name
- a multiplier
- a minimum useful time
- tracked time spent on it

## Relationship

- one **Category** contains many **Activities**
- one **Activity** belongs to one **Category**

## Purpose of the model

This model supports the main app behavior:
- the user defines what they want to do
- the user defines the relative importance of categories and activities
- the app tracks time spent locally
- the app recommends the activity that is currently most behind

## Storage model

The current plan is local-first:

- the app is implemented in Flutter/Dart
- services and repositories live inside the app
- data is stored locally in SQLite
- there are no user accounts for now
- users can manually export and import their data when switching devices

This document can be expanded later with fields, rules, and recommendation logic.
