create table if not exists code_activity (
    repository varchar check (length(repository) < 128),
    workspace varchar check (length(workspace) < 64),
    "filename" varchar check (length("filename") < 64),
    "language" varchar check (length("language") < 32),
    "row" smallint check ("row" >= 0),
    "column" smallint check ("column" >= 0),
    code_chunk varchar,
    reported_at timestamp
);

create or replace view sessions as (
    with marked as (
        select
            reported_at,
            lag(reported_at, 1, reported_at) over (order by reported_at) as prev_time,
            age(reported_at, prev_time) as time_diff,
            case
                when time_diff > interval 1 hour
                then 1 else 0
            end as is_new_session,
        from code_activity
    ),
    sessioned as (
        select
            sum(is_new_session) over (order by reported_at rows unbounded preceding) as session_id,
            reported_at
        from marked
    )

    select
        session_id as id,
        min(reported_at) as "start",
        max(reported_at) as "end",
        age("end", "start") as duration
    from sessioned
    group by id
    order by "start" desc
);