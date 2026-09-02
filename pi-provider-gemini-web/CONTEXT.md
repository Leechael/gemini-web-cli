# Gemini Web Pi Provider

This extension connects Pi to one configured internal gemini-web-cli server and preserves the remote Gemini conversation across Pi turns.

## Language

**Effective configuration**:
The single server configuration selected for the current Pi session. Trusted project configuration replaces global configuration.

**Gemini chat**:
A server-side Gemini conversation identified by `chat_id`.
_Avoid_: Pi session

**Pi session**:
Pi's persisted conversation tree. It may resume one Gemini chat or fork into a new Gemini chat.
_Avoid_: Gemini chat

## Invariants

1. Project configuration is never read before Pi marks the project trusted.
2. Provider requests and Gemini research tools use the same effective server URL.
3. A resumed Pi branch continues its recorded Gemini chat.
4. The first request after a Pi fork does not mutate the parent branch's Gemini chat.
5. No authentication configuration or credential state is required.

## Configuration transitions

| State        | Trigger                                          | Source       | Next state                       | Invariant                                    |
| ------------ | ------------------------------------------------ | ------------ | -------------------------------- | -------------------------------------------- |
| unconfigured | session starts with valid global config          | Pi lifecycle | global                           | Provider and tools use the global URL        |
| unconfigured | trusted session starts with valid project config | Pi lifecycle | project                          | Provider and tools use the project URL       |
| global       | trusted session starts with project config       | Pi lifecycle | project                          | Project URL replaces global URL              |
| project      | session switches to cwd without project config   | Pi lifecycle | global or unconfigured           | Previous project URL is not retained         |
| any          | reload                                           | user         | unconfigured, global, or project | Filesystem configuration is re-read          |
| any          | invalid effective config                         | filesystem   | unconfigured                     | Provider is unavailable and error is visible |

## Chat transitions

| State          | Trigger                        | Source      | Next state     | Invariant                                              |
| -------------- | ------------------------------ | ----------- | -------------- | ------------------------------------------------------ |
| no remote chat | first model request            | user prompt | active chat    | Returned chat id is persisted in the assistant message |
| active chat    | next prompt on the same branch | user prompt | active chat    | Request carries the recorded chat id                   |
| active chat    | resume                         | user        | active chat    | Recorded chat id remains available                     |
| active chat    | fork or clone                  | user        | no remote chat | First branch request omits the parent chat id          |
