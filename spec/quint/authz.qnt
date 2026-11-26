// Ownership/authorization state machine for task operations.
// Scope: fixed task ids and two actors. A task with owner "" does not exist.
// Only the owner may complete a task; completing twice is idempotent.
module authz {
  type Outcome = { code: str, task: str }

  pure val IDS = Set("a", "b", "c")
  pure val ACTORS = Set("alice", "bob")

  var tasks: str -> str
  var done: Set[str]
  var last: Outcome

  action init = all {
    tasks' = IDS.mapBy(_ => ""),
    done' = Set(),
    last' = { code: "init", task: "" },
  }

  action create = {
    nondet id = IDS.oneOf()
    nondet owner = ACTORS.oneOf()
    all {
      tasks.get(id) == "",
      tasks' = tasks.set(id, owner),
      done' = done,
      last' = { code: "created", task: id },
    }
  }

  action completeOwner = {
    nondet id = IDS.oneOf()
    all {
      tasks.get(id) != "",
      not(done.contains(id)),
      done' = done.union(Set(id)),
      tasks' = tasks,
      last' = { code: "completed", task: id },
    }
  }

  action completeForeign = {
    nondet id = IDS.oneOf()
    nondet actor = ACTORS.oneOf()
    all {
      tasks.get(id) != "",
      tasks.get(id) != actor,
      not(done.contains(id)),
      tasks' = tasks,
      done' = done,
      last' = { code: "forbidden", task: id },
    }
  }

  // complete before create: failure leaves state unchanged.
  action completeUnknown = {
    nondet id = IDS.oneOf()
    all {
      tasks.get(id) == "",
      tasks' = tasks,
      done' = done,
      last' = { code: "not_found", task: id },
    }
  }

  // duplicate completion is idempotent in the implementation.
  action completeAgain = {
    nondet id = IDS.oneOf()
    all {
      done.contains(id),
      tasks' = tasks,
      done' = done,
      last' = { code: "completed", task: id },
    }
  }

  action step = any {
    create,
    completeOwner,
    completeForeign,
    completeUnknown,
    completeAgain,
  }

  invariant done_tasks_exist = done.forall(id => tasks.get(id) != "")
  invariant last_task_valid = last.task == "" or IDS.contains(last.task)
}
