// CSRF double-submit cookie state machine.
// Scope: a safe request issues (or keeps) a cookie token; an unsafe request
// echoes it in a form field or header. Failure and re-issue keep the cookie.
module csrf {
  type Outcome = { code: str, token: str }

  pure val TOKENS = Set("", "c1", "c2", "bad")
  pure val ISSUED = Set("c1", "c2")

  var cookie: str
  var submitted: str
  var last: Outcome

  action init = all {
    cookie' = "",
    submitted' = "",
    last' = { code: "init", token: "" },
  }

  action issue = {
    nondet token = ISSUED.oneOf()
    all {
      cookie == "",
      cookie' = token,
      submitted' = submitted,
      last' = { code: "issued", token: token },
    }
  }

  // a safe request with an existing cookie does not rotate the token.
  action issueAgain = all {
    cookie != "",
    cookie' = cookie,
    submitted' = submitted,
    last' = { code: "kept", token: cookie },
  }

  action verifyForm = {
    nondet token = TOKENS.oneOf()
    all {
      cookie != "",
      cookie' = cookie,
      submitted' = token,
      last' = { code: if (token == cookie) "allow" else "deny", token: cookie },
    }
  }

  action verifyHeader = {
    nondet token = TOKENS.oneOf()
    all {
      cookie != "",
      cookie' = cookie,
      submitted' = token,
      last' = { code: if (token == cookie) "allow" else "deny", token: cookie },
    }
  }

  // unsafe request before any safe request issued a cookie.
  action verifyBeforeIssue = {
    nondet token = TOKENS.oneOf()
    all {
      cookie == "",
      cookie' = cookie,
      submitted' = token,
      last' = { code: "deny", token: "" },
    }
  }

  action step = any {
    issue,
    issueAgain,
    verifyForm,
    verifyHeader,
    verifyBeforeIssue,
  }

  invariant allowRequiresMatch = last.code != "allow" or submitted == cookie
  invariant submittedInDomain = submitted.in(TOKENS)
}
