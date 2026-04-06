idletime
========

Problem
-------

I have too many interests and too little time.

When a time slot finally opens up for me, there are 2 issues:

* I feel overwhelmed by all the things I could spend this time frame on.
* I don't feel progress in my interests because I only do one of them for a short time.

``idletime`` addresses both.

Solution
--------

It lets you input all your interests and activities. It lets you assign how much you want to do each activity in relation to the others.
The app will tell you what activity to do right now. When you are done, it will add the time spent to that activity.
Then you can look at a graph and see your progress.

Example
-------

Setup
~~~~~

I have many interests I want to keep track of.

* For **sports**, I like to do: **Swimming**, **Jogging**, **Gym**.
* For **self-care**, I like to do: **Reading**, **Bathing**, **Journaling**.
* I want to learn: **Spanish**, **Italian**.

I want to do **2 times** as much **swimming** as I do **jogging**. I also want to go to the **gym** **1.5 times** as much as I go **jogging**.

So I input the following activities into the app:

* ``swimming = 2.0``
* ``jogging = 1.0``
* ``gym = 1.5``

I can also assign ``minimal_time`` values to these activities. Which means: when I go swimming, I need at least 30 minutes to do so. Drive there, change, shower... So I could not go swimming in a 20-minute time window. The app needs to know that. So I assign ``swimming`` a ``minimal_time`` of ``20 minutes``.

Usage
~~~~~

When I now start the app and tell it the size of my time frame, it will recommend whatever activity I currently should do in order for my activities to stay in the relation that I assigned to them.

Now I can start a timer, and after I stop it, the app will automatically add this time to the activity. I can also choose to input the time manually, or, when I do not want to do this activity right now, to ``skip`` and be recommended a different activity.

Now, I do not know whether I want to do **Sports**, **Self-Care**, or **Languages**. Which is why earlier, I assigned each of these categories a multiplier as well. I assigned a **2.0 multiplier to Sports**, a **1.0 multiplier to Self-Care**, and a **0.3 multiplier to Languages**. This way I do three times as much self-care as I learn languages, and twice as much sports as I do self-care.

Now I can tell the app to give me an activity out of all of these categories. It will then give me an activity.

Current project status
~~~~~~~~~~~~~~~~~~~~~~

As of 2026-04-06, the first working prototype already includes:

* a React frontend in ``frontend/app`` and a Go backend in ``backend``
* user accounts with email verification and session-cookie authentication
* user-scoped CRUD for categories, activities, and time entries
* a first recommendation flow that returns the single best activity based on how far behind it is relative to the configured multipliers
* stateless ``skip`` support for asking for the next best recommendation
* a default ``root`` category so users do not have to organize everything into custom categories

Still planned for later:

* use available time windows and ``minimum_minutes`` more directly in the recommendation logic
* add timer and graph/analytics views
* continue polishing the frontend UX
