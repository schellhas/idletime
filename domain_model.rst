Domain Model
============

This file describes the core objects of ``idletime`` and how they relate to each other.

Core objects
------------

The current runtime model is still intentionally small, but it is now multi-user and consists of four main persisted objects:

* **User**
* **Category**
* **Activity**
* **TimeEntry**

A recommendation is derived from these objects at request time instead of being stored permanently.

User
----

A user owns all of their own categories, activities, and time entries.

Each newly registered user also gets a default ``root`` category automatically so the app can still be used without defining custom categories first.

Category
--------

A category groups activities together.
The built-in ``root`` category acts as the fallback category for users who do not want to organize their activities further.

Examples:

* ``sport``
* ``self-care``
* ``languages``
* ``leisure``
* ``productivity``

A category can also have its own multiplier so that some areas of life are prioritized more than others.

Activity
--------

An activity belongs to a category.

Examples:

* ``swimming``
* ``jogging``
* ``gym``
* ``reading``
* ``journaling``
* ``spanish``

Each activity can have:

* a name
* a multiplier
* a minimum useful time
* tracked time spent on it

TimeEntry
---------

A time entry records one logged duration for an activity.
It stores the minutes spent, an optional note, and the creation timestamp.
Updating time entries changes the tracked progress that later feeds into recommendations.

Relationship
------------

* one **User** owns many **Categories**
* one **Category** contains many **Activities**
* one **Activity** belongs to one **Category**
* one **Activity** has many **TimeEntries**
* one **TimeEntry** belongs to one **Activity**
* a recommendation is computed from tracked time and the combined category/activity multipliers

Purpose of the model
--------------------

This model supports the main app behavior:

* the user defines what they want to do
* the user defines the relative importance of categories and activities
* the app tracks time spent
* the app recommends the activity that is currently most behind

Storage model
-------------

The current plan is a frontend/backend split:

* the frontend is implemented in **React**
* the backend is implemented in **Go**
* data is stored in **PostgreSQL**
* business logic and recommendation logic live in the backend
* multi-user accounts and session-cookie auth are implemented
* each user gets a default ``root`` category so categorization stays optional

This document can be expanded later with fields, rules, and recommendation logic.
