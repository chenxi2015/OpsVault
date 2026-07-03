# OpsVault TUI Operations Workbench Design

## Context

OpsVault already has a working CLI command tree and driver layer, plus a basic TUI shell under `internal/tui`. The current TUI is still read-only in practice: it periodically loads `driver.ServiceStatus` data and renders string views for Dashboard, Nginx, Docker, and Config panels.

The next step is not a visual refresh. It is to turn the TUI into an operational workbench that lets an operator inspect services, trigger actions, see task progress, and confirm results without dropping back to the CLI for common workflows.

This design keeps the existing architectural constraints:

- `driver.ServiceDriver` remains the fixed lifecycle interface.
- TUI remains decoupled from deployment logic and reuses driver-layer behavior.
- Docker services stay on the Docker driver path.
- Nginx stays on the binary driver path.

## Goal

Deliver a first interactive TUI iteration that feels like an operations console rather than a status dashboard.

The first release of this workbench should support:

- direct service operations for Docker-managed middleware
- direct service operations for Nginx
- Nginx vhost and SSL high-frequency workflows
- visible task execution feedback
- automatic status refresh after actions

It should not attempt to solve every future workflow in the first pass.

## Product Direction

The approved direction is:

- main shape: workbench
- execution model: direct actions with confirmation only for dangerous operations
- first functional scope: both Docker and Nginx, but only high-frequency actions

This favors fast daily operations over wizard-heavy flows.

## UX Model

The TUI should move from simple tab content to a shell with three persistent concerns:

1. selection
2. action
3. feedback

The shell layout is:

- top tab bar: `Dashboard`, `Nginx`, `Docker`, `Config`
- left rail: list of services or resources within the active panel
- right detail area: status, metadata, and actions for the selected item
- bottom drawer: tasks, output, and logs

The core interaction loop is:

1. choose target
2. trigger action
3. watch task feedback
4. observe refreshed state

## Panel Design

### Dashboard

Dashboard is a summary and launch surface, not a second full management screen.

It shows all services in a compact table and supports:

- jump to service-specific panel
- high-frequency actions on the selected row:
  - start
  - stop
  - restart
  - logs

Dashboard should not host long forms or detailed configuration editing.

### Docker Panel

The Docker panel is service-centric.

Left rail:

- mysql
- redis
- rocketmq
- rabbitmq
- postgres

Right detail area shows:

- running state
- image tag
- network
- ports
- data path
- storage usage when available
- last refresh time
- last action result

Primary actions:

- install
- start
- stop
- restart
- upgrade
- uninstall
- logs

Rules:

- actions should reflect current service state
- dangerous actions require confirmation
- task drawer opens automatically when an action starts

### Nginx Panel

The Nginx panel is resource-centric, not only service-centric.

Top-level resource groups in the left rail:

- Service
- VHosts
- Certificates

When `Service` is selected, the right side shows:

- running state
- version
- PID
- install path
- `www_root`
- `ssl_root`

Primary actions:

- install
- start
- stop
- restart
- reload
- upgrade
- uninstall
- logs

When `VHosts` is selected, the left rail becomes the vhost list and the right side shows:

- domain
- root path
- SSL enabled or disabled
- related config path if available

Primary actions:

- add
- delete
- enable SSL
- disable SSL

When `Certificates` is selected, the left rail becomes the certificate/domain list and the right side shows certificate-specific metadata when available.

Primary actions:

- apply
- renew
- delete

The first release may use minimal certificate metadata if full inspection is not yet available from the driver layer.

### Config Panel

The config panel is not part of the first interactive milestone beyond remaining present as a placeholder surface. It should not block the workbench rollout.

## Interaction States

The TUI should use a small explicit state model:

- `idle`
- `loading`
- `acting`
- `confirming`
- `editing`

Rules:

- only the current target's repeated action controls should lock during `acting`
- global navigation remains available during long-running tasks
- dangerous actions transition through `confirming`
- successful form submission returns to the target detail view
- every completed action triggers status refresh

## Dangerous Actions

The first release must confirm these operations:

- uninstall
- uninstall with purge
- delete vhost with `delete-root`
- SSL delete

Confirmation UI can be lightweight. A simple modal or inline confirmation layer is enough for the first pass.

## Action Scope for the First Milestone

### In Scope

- Docker lifecycle actions:
  - install
  - start
  - stop
  - restart
  - upgrade
  - uninstall
  - status refresh
  - logs
- Nginx lifecycle actions:
  - install
  - start
  - stop
  - restart
  - reload
  - upgrade
  - uninstall
  - status refresh
  - logs
- Nginx resource actions:
  - vhost add
  - vhost delete
  - vhost list
  - SSL apply
  - SSL renew
  - SSL delete
- bottom drawer for:
  - task list
  - current task output
  - selected service logs

### Out of Scope

- bulk operations
- advanced configuration editing
- full config wizard authoring flow
- rich markdown rendering work
- multi-pane concurrent log viewers
- large install parameter editors

## Driver and Capability Design

`driver.ServiceDriver` stays unchanged and remains the minimum common contract.

To support an interactive TUI without hard-coding service-specific logic into the panels, introduce optional capability interfaces layered on top of `ServiceDriver`.

Expected capabilities include:

- log reading
- reload support
- Nginx vhost management
- Nginx SSL management
- service-specific informational actions such as RocketMQ version or DLQ statistics in later iterations

The TUI should discover capabilities from the service object rather than assume them from the service name alone.

## Driver Refactoring Direction

The TUI should not call Cobra commands.

Behavior currently assembled in command handlers should move into driver-facing methods where needed, especially:

- Nginx SSL flows that currently combine `sslutil` work with Nginx config switching
- service log retrieval that is currently implemented inside CLI commands

This keeps the TUI dependent on business interfaces rather than shelling back through the command layer.

## Task Execution Model

The TUI needs an internal task runner that converts user intent into async work and emits structured task updates.

Required task event types:

- task started
- task output appended
- task succeeded
- task failed

The runner should execute driver actions and return messages back into the Bubble Tea update loop.

The first release does not need a general-purpose job scheduler. It only needs enough structure to:

- keep the UI responsive during work
- show progress or output
- serialize feedback into the drawer

## Logs Model

For the first release, log support can be "recent lines on demand" instead of full continuous streaming.

This is sufficient to make the workbench useful while keeping implementation risk controlled.

Suggested minimum:

- request latest N lines from Docker services
- request latest N lines from Nginx via `journalctl`
- render output in the bottom drawer
- allow refresh and close

True live tail behavior can be added later without invalidating this design.

## TUI Internal Structure

Refactor the TUI into a shell plus panel models.

### Root Model Responsibilities

`RootModel` should own:

- active tab
- global focus region
- window size
- shared task drawer state
- global refresh scheduling
- dispatch of task messages

### Panel Model Responsibilities

Each panel should become a real `tea.Model`-like unit with its own selection and local interaction state.

Target panel units:

- Dashboard panel
- Docker panel
- Nginx panel
- Config panel

Each panel owns:

- selected item
- locally available actions
- form or confirmation state
- panel-local error state

## Service Registry Direction

The current TUI status provider directly constructs drivers to collect statuses.

The interactive version should move toward a service registry abstraction that can provide:

- service identity
- driver reference
- status
- capability discovery
- available actions

This keeps panel logic simpler and creates a single integration point between TUI and driver-layer services.

## Suggested Key Bindings

The first release should prioritize fast keyboard operation:

- `Tab`: switch focus region
- `Up` / `Down`: move selection
- `Enter`: open or confirm primary action
- `s`: start
- `x`: stop
- `r`: restart
- `u`: upgrade
- `i`: install
- `d`: open dangerous action confirmation
- `l`: open logs drawer
- `t`: open tasks drawer
- `/`: filter or search
- `esc`: close modal, form, or drawer layer

Bindings may evolve, but the first plan should treat fast action dispatch as a design requirement, not a nice-to-have.

## Delivery Plan

### Phase 1: Workbench Shell

Build the interactive shell without full business depth:

- convert static panel views into interactive panel models
- add focus management
- add selection state
- add bottom drawer
- add task status rendering

### Phase 2: Docker Service Operations

Make the Docker panel truly usable:

- capability-backed actions
- confirmation flow
- task execution
- recent-log viewing
- post-action refresh

### Phase 3: Nginx Resource Operations

Extend the same operating model to Nginx service and Nginx resources:

- reload support
- vhost forms
- SSL forms
- recent-log viewing
- confirmation flow

## Risks and Design Constraints

The main planning risks are:

- over-building UI chrome before action flows work
- leaking Cobra command logic into the TUI
- forcing Nginx resource operations through the generic lifecycle interface
- attempting streaming logs and full form systems too early

The plan should prefer operational closure over visual sophistication.

## Planning Notes

The implementation plan should preserve these decisions:

- do not change the fixed `ServiceDriver` lifecycle contract
- add optional capability interfaces rather than widen the fixed base interface
- keep first-release logs simple
- prioritize Docker lifecycle operations before Nginx resource depth if sequencing pressure appears
- do not block the workbench milestone on the Config panel

## Open Implementation Questions

These questions should be answered during planning, not by changing the product direction:

- exact shape and file placement of capability interfaces
- exact task event message types
- whether panel models share a common interface or only common conventions
- whether logs are returned as strings first or via readers from day one

These are implementation decisions inside the approved product shape, not unresolved product requirements.
