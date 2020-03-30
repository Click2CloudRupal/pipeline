CREATE TABLE "public"."processes" (
    "id" text NOT NULL,
    "parent_id" text,
    "org_id" int4 NOT NULL,
    "name" text NOT NULL,
    "type" text NOT NULL,
    "resource_id" text NOT NULL,
    "status" text NOT NULL,
    "started_at" timestamptz NOT NULL DEFAULT now(),
    "finished_at" timestamptz NOT NULL DEFAULT '1970-01-01 00:00:01+00'::timestamp with time zone,
    PRIMARY KEY ("id")
);

CREATE INDEX idx_start_time_and_time ON "processes"("started_at", "finished_at");

CREATE TABLE "public"."process_events" (
    "process_id" text,
    "log" text NOT NULL,
    "name" text NOT NULL,
    "status" text NOT NULL,
    "timestamp" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "process_events_process_id_processes_id_foreign" FOREIGN KEY ("process_id") REFERENCES "public"."processes"("id") ON DELETE RESTRICT ON UPDATE RESTRICT
);
