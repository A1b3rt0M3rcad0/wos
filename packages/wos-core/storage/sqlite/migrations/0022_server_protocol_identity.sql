-- Immutable instance identity survives restore and process restarts.
-- Secret signing material is deliberately absent.
CREATE TABLE server_protocol_identity (
 singleton INTEGER PRIMARY KEY CHECK(singleton=1),
 state_json TEXT NOT NULL
);
