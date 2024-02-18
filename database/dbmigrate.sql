CREATE TABLE public.tasks
(
    task_id bigserial NOT NULL,
    data jsonb,
    PRIMARY KEY (task_id)
);

ALTER TABLE IF EXISTS public.tasks
    OWNER to postgres;
