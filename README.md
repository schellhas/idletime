# idletime

## Advertisement-style description

I have too many interests — and maybe you do too.

Then, when you suddenly have `30 minutes` or `40 minutes`, you may not know what to spend it on. Not because you have nothing to do, but because you care about too many things at once: sports, leisure, productivity, learning, creativity. In a small timeframe, doing something random can feel pointless. Will you ever really see progress this way?

`idletime` is the app for that problem.

It helps you use these small pockets of time with confidence. You can continuously add small amounts of time to the things you care about, see that the time is not lost, and make sure you are actually doing your activities as much as you want to in relation to each other.

The idea is simple: first, you add the activities you like. Then you assign them multipliers.

Let us say you want to stay fit. Your favorite sports are `climbing`, `swimming`, and `muay thai`.

You could set:
- `swimming = 1.0`
- `climbing = 1.5`
- `muay thai = 2.0`

That means you want to do twice as much muay thai as swimming, and 1.5 times as much climbing as swimming.

Now, when you say:

> `I have 40 minutes, give me an activity.`

The app looks at how much time you have already logged for each activity and compares that against the multipliers you chose. In other words, it keeps track of which activity is currently most behind its desired share. Then it recommends the activity with the largest deficit.

This way, the app does not just give you something random. It gives you the thing that makes the most sense for your longer-term balance.

The app also respects minimum time requirements. For example, `swimming` might have a minimum time because going there, changing clothes, showering, and coming back makes `5 minutes` meaningless. So when you first add an activity, you can define its minimum useful time. If you only have `5 minutes`, the app will skip swimming and recommend something else.

And it does not stop at sport.

You can also create other categories, for example `leisure`, and put things like `reading`, `drawing`, and `making music` in there, again with the multipliers you want.

If you do not even know whether you want to do `sport` or `leisure`, you can simply tell the app to choose from either category. It will then look across all allowed activities and again recommend the one with the largest deficit.

Once the app gives you an activity, you can start a timer. When you pause the timer, the app lets you conveniently add the saved time to that activity. You can also adjust the time if something got in the way. If you do not want that suggestion, you can choose `recommend something else` and get the next-most-deficient activity.

If you already know what you want to do, you can simply add the time manually. The app will take that into account the next time it calculates which activity is most behind.

## Neutral description

`idletime` is an app that helps choose what to do with the time available right now while balancing multiple interests over the long term.

The user defines:
- categories such as `sport`, `leisure`, or `productivity`
- activities inside those categories
- multipliers that express how much they want to do one activity or category relative to another
- a minimum useful time for each activity

The app keeps track of time spent on each activity, either through a timer or by manual input.

When the user asks for a suggestion, the app filters activities by the selected categories and by whether the available time is enough for that activity. It then compares the logged time against the configured multipliers and recommends the activity with the largest deficit relative to its target share.

After getting a suggestion, the user can choose `recommend something else`, start a timer for it, or log time manually. The user can also log time directly for any activity without using the recommendation flow first.