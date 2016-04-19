# Task schedule optimizer web service

This is a web service written to optimally schedule a set of work tasks
given the following:
- A list of tasks to accomplish each with a business value ("reward") and
  estimated time to complete in hours. Tasks may optionally also have a deadline
  and a minimum start date.
- Weekly work time blocks
- Appointments when project work cannot be scheduled
- Time zone

## Request sample and format explanation

Here's a sample post which will inform the explanation of each piece of it:

```
POST /

{
  "timeZone": "America/New_York",
  "weeklyTaskBlocks": [
    [],
    [{"start": "10:00", "end": "12:00"}],
    [{"start": "9:00", "end": "10:00"}, {"start": "11:30", "end": "14:30"}],
    [],
    [],
    [{"start": "16:00", "end": "18:00"}],
    []
  ],
  "appointments": [
      {"title": "Meeting", "start": "2016-04-17T16:00:00Z", "end": "2016-04-17T18:00:00Z"}
  ],
  "tasks": [
    {"title": "Newsletter", "estimatedHours": 3, "reward": 9, "deadline": "2016-04-20T22:00:00Z", "startOnOrAfter": "2016-02-17T15:00:00Z"},
    {"title": "Reimbursement", "estimatedHours": 2, "reward": 15, "startOnOrAfter": "2015-02-18T15:00:00Z"}
  ],
  "startTaskSchedule": "2015-02-16T14:00:00Z",
  "endTaskSchedule": "2015-02-28T22:00:00Z"
}
```

The `timeZone` should be the name of the time zone of the weekly time work
blocks, e.g. "America/New_York".

The `weeklyTasksBlocks` is an array of 7 arrays. Each of the 7 lists correspond
to the generally available project work blocks in a given week. List 0
corresponds to Sunday, list 1 to Monday, etc. So in the example above, that
represents having weekly project work availability on Monday from 10am-12pm, and
then on Tuesday from 9am-10am and 11:30am-2:30pm, and Friday from 4pm-6pm. Note
that the times are given in a 24 hour clock format. This might correspond to
weekly normal working hours, for instance.

Next `appointments` is a list of time periods that project work cannot be
scheduled even if that appointment falls during one of the normal weekly time
blocks that could be used for project work.

Then comes the `tasks` list. Each task should have a `title` field which
describes the task, then `estimatedHours` and `reward` (a measure of the
business value of the task). Currently `estimatedHours` must be a whole number
of hours. A task may optionally also have `deadline`,
`startOnOrAfter`, or both of those fields. A task represents a project you want
to accomplish during those weekly project work hours.

Finally, the `startTaskSchedule` and `endTaskSchedule` give the start and end
times for the calculation to take place over. Often `startTaskSchedule` will be
the current time and `endTaskSchedule` should be far enough into the future to
allow for scheduling all the tasks assuming there are no conflicts with
deadlines.

Here's the response our sample request would give:
```
[
  {
    "title": "Newsletter",
    "start": "2015-02-16T15:00:00Z",
    "end": "2015-02-16T17:00:00Z",
    "finish": false
  },
  {
    "title": "Newsletter",
    "start": "2015-02-17T14:00:00Z",
    "end": "2015-02-17T15:00:00Z",
    "finish": true
  },
  {
    "title": "Reimbursement",
    "start": "2015-02-20T21:00:00Z",
    "end": "2015-02-20T23:00:00Z",
    "finish": true
  }
]
```

What it gives is an optimal arrangement of the into chunks of time (minimal size
of 1 hour) that will fit into the weekly time blocks and work around the
specified appointments. They could then be auto-added to a calendar easily.
There is one additional field `finish` for each task that indicates whether that
is the final (finishing) work block for that particular task or whether there
will still be more work blocks on it to come.

If a deadline cannot be met the service will respond with:
`{"err":"Could not solve linear program"}`

## How the optimization works

Here's a general overview of how the scheduler works.

First it has an initial step where it considers all the weekly time blocks and
the appointments and maps them to a flat list of "available work hours" and when
those available work hours would be. It also translates the deadlines and
minimum start dates into indexes into that "available work hours" list.

Next it forms a linear program as follows:
- Each task will have a set of variables that correspond to all of the different
  hours that task could be scheduled on. I.e. there are
  `available work hours * number of tasks` total variables of the form
  `task[task_index]_hour[hour_index]`. If variable `task[0]_hour[0]` is 1 then
  that means that you will do task 0 at hour 0.
- Each variable has a constraint that it must be at minimum 0 and maximum 1.
- All the task variables for a particular hour have the constraint that they
  must sum to at most 1 - i.e. you can't schedule more than one hour of tasks on
  a given hour.
- The sum of all the hours of a given task must be less than or equal to the
  estimated hours for that task, i.e. you can't keep doing a valuable task over
  and over after it's done but can only capture its business value once.
- The deadline for a given task is modeled as a constraint that the sum of all
  the scheduled hours for a given task before the deadline be the total
  estimated time of the task.
- Likewise, a minimum start time is a constraint that the total hours of the
  task before the start time be zero.

The objective function of the linear program is the sum of all the `reward/hour`
for each task multiplied by all the hours that task is scheduled.

Because this just uses a linear program with continuous variables, how does this
avoid fractional assignments of the variables? Well, assuming the `reward/hour`
of each task is different, and the tasks have whole numbers of estimated hours,
then the tasks will naturally pack into full hour slots to maximize the
objective function. Because ties in the `reward/hour` of different tasks is
possible, the program adds a small random "nudge" value to that hourly reward to
pre-emptively break the ties. The random nudging is done with a fixed seed so it
is consistent for a given input.

Internally this uses the [golp](https://github.com/draffensperger/golp) library
which wraps the [LPSolve](http://lpsolve.sourceforge.net/5.5/)linear programming
solver.

## How to use it

Currently in my personal use of this for my own schedule I use Google sheets
with a [Google apps script](https://gist.github.com/draffensperger/039ca1834b03cb49c551eaa34d5abb7c) as the front-end for this service.

I have started on a [web front end](https://github.com/draffensperger/wizweek)
for it but haven't gotten very far with it.

Feel free to use it to build your own personal task scheduling system as well!

## Deployment

This has been set up to be easily deployed to Heroku as the lpsolve55.so file is
bundled with the code and there is an included Heroku `Procfile` which specifies
how to run the Go service.

## License and Acknowledgements

This idea of optimizing your tasks is based on an Excel spreadsheet my dad, John
Raffensperger users to manging his tasks. For his explanation of it and a link
to his spreadsheet, see [john.raffensperger.org/](http://john.raffensperger.org/).

The bundled `lpsolve55.so` file is [LGPL licensed](http://lpsolve.sourceforge.net/5.0/LGPL.htm).

The Go code in this project is MIT licensed as follows:

MIT License (MIT)

Copyright (c) 2015 David Raffensperger

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
