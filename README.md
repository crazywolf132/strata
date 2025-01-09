# Strata

The Git workflow you wish you'd always had.

## Why Strata?

Are you drowning in endless manual rebases, merge conflicts, and giant PRs that your teammates dread reviewing? Tired of your Git logs looking like someone lost a game of Jenga? Strata swoops in to save the day with a more elegant, stacked approach to your Git workflow—so you can stay focused on shipping great features and banish that Git friction once and for all.

## The Problem

### Big PRs, Big Headaches
- Time-consuming reviews: Large, monolithic pull requests scare away reviewers, making your team's velocity plummet.
- Forever merges: You can't ship your next feature until the current PR is merged—leading to queues of branches stuck in code-review purgatory.
- Git confusion: Traditional branching is powerful, but it's easy to get tangled in rebase merges, conflict storms, and uncertain dependencies.

### Delays and Bottlenecks
- You spend countless hours waiting on a single PR to land before even starting the next.
- Each time you push a fix for the old PR, you trigger conflict cascades in your new feature branches.
- You're under the clock to deliver, yet half your day is spent wrangling merges, not writing code.

## How Strata Solves It

### Stacked PRs—Made Simple

Strata introduces the concept of stacked changes: each feature or fix is a tidy branch that builds on the last—no waiting required. Want to start your front-end changes before the backend merges? Go right ahead. Our intuitive CLI does the heavy lifting.

### Offline-First & Optional Daemon
- **Offline freedom**: No server needed. Work on a plane, on top of a mountain, or anywhere with flaky Wi-Fi.
- **Optional daemon**: Turn it on for auto-sync and collaboration. If you don't need it, Strata still works flawlessly with your local .git folder.

### Collaboration for the Real World
- **Share code**: Easily hand off partial work to a teammate using a simple share code—boom, they've got your entire stack.
- **Enterprise server (optional)**: Securely host your team's ephemeral stack data if local sharing isn't enough.
- **CI gating**: Our built-in ci check command ensures your stack meets your org's merge rules before hitting production.

### Smart Rebasing & Merging
- **Transaction-like safety**: We tag your branch before merges or rebases, so if something goes wrong, Strata reverts automatically—no more "uh-oh, lost my commits" horror stories.
- **Auto conflict resolution options**: Choose a policy (ours, theirs, or manual prompts) to handle merges quickly.

### Powerful Yet Fun to Use
- **Minimal cognitive load**: Familiar Git commands, but wrapped in a simpler mental model.
- **Hooks & customization**: Launch scripts on events like "layer created," "stack updated," etc. Use them for CI, linting, or rocket-launch macros.
- **Developer delight**: Once you experience stacked changes, you'll never want to manage monstrous single-branch PRs again.

## Key Features

1. **Stacked Layers**: Break up big changes into small, comprehensible branches—Strata manages the dependencies, so you don't have to.
2. **Merge & Rebase Magic**: Automatic conflict detection, transaction-like merges, and conflict resolution policies.
3. **Optional Collaboration**: Generate a share code to let your colleague jump in and help. Use a central server if you want enterprise-level team sharing.
4. **CI Integration**: Gate merges with strata ci check, ensuring your branch passes all the rules before shipping.
5. **Hooks**: Automate tasks before or after merges, rebases, or new layer creation.
6. **Config Where You Expect**: Global config in ~/.config/strata, local config in your repo, or environment variables—flexible and consistent.

## Getting Started

### 1. Install & Initialize

```bash
go install github.com/crazywolf132/strata@latest
# or clone and build
git clone https://github.com/crazywolf132/strata.git
cd strata
go build -o strata .

cd /path/to/repo
git init
strata init
```

### 2. Add Your First Layer

```bash
strata add feature-1
# Write some code...
git commit -m "Implement feature-1"
strata push
```

### 3. View & Merge

```bash
strata view   # see your layered stack
strata merge feature-1
```

### 4. Enjoy Freedoms You Didn't Know Git Could Offer.

## Usage Highlights

- `strata add <branch>`: Create a new stacked layer on top of your current branch.
- `strata update`: Rebase each branch onto its parent. No more manual rebase nightmares.
- `strata share`: Generate a code for your coworker to clone your entire stack.
- `strata use <code>`: Pull someone else's shared stack for parallel dev.
- `strata ci check <branch>`: Validate a branch's merge feasibility (great for pipelines).
- `strata daemon`: Optional background process for auto-sync.

## When to Use Strata

- **Large Features**: Breaking down a huge feature into micro-layers that are easier to review.
- **Parallel Development**: Start front-end dev before the backend merges, or vice versa.
- **Team Collaboration**: Onboard a new teammate mid-sprint by sending them a quick share code.
- **Offline**: Work from the coffee shop or airplane seat—commit, rename, rebase, all offline, no regrets.

## Who is Strata For?

- **Solo Developers**: Enjoy simpler merges and PR flows without needing any server overhead.
- **Small Teams**: Share partial work, keep PRs small, and ship faster.
- **Enterprises**: Self-host an on-prem Strata server for secure collaboration, while your devs remain frictionless.

## Contributing

Strata is open source—pull requests, issues, and feature ideas are welcome. We'd love your help polishing the diamond or forging new edges. Check out our contribution guidelines for how to get started.

## License

MIT License – Because we believe in open collaboration, synergy, and the unstoppable force of community-driven dev tools.

---

Embrace the stacked life. Once you've tasted small, incremental PRs that flow like water, you'll never want to fight a monstrous single-branch merge again. Let Strata handle the branching overhead while you focus on building awesome software.

Happy stacking!
