# 🌙⚙️ Midnight Runner

## Dependencies

- [github.com/jessevdk/go-flags](https://github.com/jessevdk/go-flags)
- [github.com/reugn/go-quartz](https://github.com/reugn/go-quartz)

## Manage

![1](pics/base.png)

![3](pics/setjob.png)

![2](pics/logs.png)

## Cron expression format

| Field Name   | Mandatory | Allowed Values  | Allowed Special Characters |
|--------------|-----------|-----------------|----------------------------|
| Seconds      | YES       | 0-59            | , - * /                    |
| Minutes      | YES       | 0-59            | , - * /                    |
| Hours        | YES       | 0-23            | , - * /                    |
| Day of month | YES       | 1-31            | , - * ? / L W              |
| Month        | YES       | 1-12 or JAN-DEC | , - * /                    |
| Day of week  | YES       | 1-7 or SUN-SAT  | , - * ? / L #              |
| Year         | NO        | empty, 1970-    | , - * /                    |

### Special characters

- `*`: All values in a field (e.g., `*` in minutes = "every minute").
- `?`: No specific value; use when specifying one of two related fields (e.g., "10" in day-of- month, `?` in
  day-of-week).
- `-`: Range of values (e.g., `10-12` in hour = "hours 10, 11, and 12").
- `,`: List of values (e.g., `MON,WED,FRI` in day-of-week = "Monday, Wednesday, Friday").
- `/`: Increments (e.g., `0/15` in seconds = "0, 15, 30, 45"; `1/3` in day-of-month = "every 3 days from the 1st").
- `L`: Last day; meaning varies by field. Ranges or lists are not allowed with `L`.
  - Day-of-month: Last day of the month (e.g, `L-3` is the third to last day of the month).
  - Day-of-week: Last day of the week (7 or SAT) when alone; "last xxx day" when used after
    another value (e.g., `6L` = "last Friday").
- `W`: Nearest weekday in the month to the given day (e.g., `15W` = "nearest weekday to the 15th"). If `1W` on
  Saturday, it fires Monday the 3rd. `W` only applies to a single day, not ranges or lists.
- `#`: Nth weekday of the month (e.g., `6#3` = "third Friday"; `2#1` = "first Monday"). Firing does not occur if
  that nth weekday does not exist in the month.

<sup>1</sup> The `L` and `W` characters can also be combined in the day-of-month field to yield `LW`, which
translates to "last weekday of the month".

<sup>2</sup> The names of months and days of the week are not case-sensitive. MON is the same as mon.
