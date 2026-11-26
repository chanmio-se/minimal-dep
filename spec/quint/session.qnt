// Session create / update / expiry / authentication state machine.
// Scope: abstract tokens (t1..t3) and subjects (alice/bob). A token binding is
// either a subject or "" (invalidated). There is no clock: expiry and logout
// are modeled as the same "invalidate" event. Rotation replaces a live token
// with a fresh one for the same subject.
module session {
  type Outcome = { code: str, subject: str }

  pure val TOKENS = Set("t1", "t2", "t3")
  pure val SUBJECTS = Set("alice", "bob")

  var sessions: str -> str
  var last: Outcome

  action init = all {
    sessions' = TOKENS.mapBy(_ => ""),
    last' = { code: "init", subject: "" },
  }

  // create or update: bind a fresh token to a subject.
  action issue = {
    nondet token = TOKENS.oneOf()
    nondet subject = SUBJECTS.oneOf()
    all {
      sessions.get(token) == "",
      sessions' = sessions.set(token, subject),
      last' = { code: "issued", subject: subject },
    }
  }

  action authenticate = {
    nondet token = TOKENS.oneOf()
    all {
      sessions.get(token) != "",
      sessions' = sessions,
      last' = { code: "authenticated", subject: sessions.get(token) },
    }
  }

  // authenticate before create, after expiry, or with an unissued token.
  action authenticateUnknown = {
    nondet token = TOKENS.oneOf()
    all {
      sessions.get(token) == "",
      sessions' = sessions,
      last' = { code: "unknown", subject: "" },
    }
  }

  // update/rotation: replace a live token with a fresh one for the same subject.
  action rotate = {
    nondet previous = TOKENS.oneOf()
    nondet token = TOKENS.oneOf()
    nondet subject = SUBJECTS.oneOf()
    all {
      sessions.get(previous) == subject,
      sessions.get(token) == "",
      sessions' = sessions.set(previous, "").set(token, subject),
      last' = { code: "rotated", subject: subject },
    }
  }

  action expire = {
    nondet token = TOKENS.oneOf()
    all {
      sessions.get(token) != "",
      sessions' = sessions.set(token, ""),
      last' = { code: "expired", subject: "" },
    }
  }

  action logout = {
    nondet token = TOKENS.oneOf()
    all {
      sessions.get(token) != "",
      sessions' = sessions.set(token, ""),
      last' = { code: "logged_out", subject: "" },
    }
  }

  action step = any {
    issue,
    rotate,
    authenticate,
    authenticateUnknown,
    expire,
    logout,
  }

  invariant valid_bindings =
    TOKENS.forall(t => sessions.get(t) == "" or SUBJECTS.contains(sessions.get(t)))

  invariant last_subject_valid = last.subject == "" or SUBJECTS.contains(last.subject)
}
