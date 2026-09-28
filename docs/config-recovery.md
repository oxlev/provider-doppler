# Config recovery investigation

## Outcome

No additional recovery code is justified by the available evidence. Keep the
provider fail-closed on ambiguous errors rather than silently equating lost
access with deletion. This does not block the tested creation/reference fixes.

The standard Terraform/Crossplane recovery path works when Doppler returns 404
and management policies allow Create. Recovery is not guaranteed when the API
returns an ambiguous access error for a missing config.

## Live observations

- The original missing-config read returned HTTP 400 with
  `This token does not have access to requested config`. Terraform import failed,
  but creation from empty Terraform state succeeded with the same token.
- On follow-up, the previously missing `ci_fixed` config returned HTTP 404 with
  `Could not find requested config`. The credential file matched the installed
  provider credential (compared without displaying either value).
- The paused, annotated test Config was resumed without resetting its external
  identity. It reached Ready through the normal Terraform read/create path.
- We did not delete a previously created config for this follow-up. This validates
  recovery from a missing annotated identity with a 404, not the complete
  external-deletion lifecycle or all token/permission combinations.
- We did not establish why the response changed. Matching token bytes do not
  prove unchanged server-side permissions or API behavior. Do not claim a Doppler
  fix or universal 404 semantics based on this result.

Only read calls and resumption of the existing disposable test resource were
used for the API investigation. No permission changes or external deletions were
performed.

## Can listing prove absence?

The [config list reference](https://docs.doppler.com/reference/configs-list)
documents project, optional environment, page, and per_page, but does not state
that a successful response includes objects the caller cannot access. The
[retrieve reference](https://docs.doppler.com/reference/configs-get) does not
specify a reliable distinction between missing and unauthorized configs.

Doppler supports restricted access:

- [Project permissions](https://docs.doppler.com/docs/project-permissions)
- [Service accounts](https://docs.doppler.com/docs/service-accounts)
- [Config-scoped service tokens](https://docs.doppler.com/docs/service-tokens)

With the test token:

- Project-only listing with `per_page=1` returned one config per page and then
  an empty page (six configs across six nonempty pages after recovery).
- Environment-filtered listing with `per_page=1` returned all five then-visible
  configs on page 1; page 2 returned HTTP 400, `Page must be 1 when environment
  is specified`. A generic pagination assumption is therefore insufficient.

This establishes observed pagination behavior, not completeness across access
restrictions. We did not create new credentials or revoke access just to probe
visibility. A list-based fallback remains unsafe without an authoritative API
contract covering permissions, filtering, pagination, and consistency.

## Upstream reproduction / support request draft

Suggested title: **Missing config read can return ambiguous access error and block Terraform refresh**

Environment: Terraform 1.5.7 and Doppler Terraform provider 1.21.5. Use a disposable
project/environment and a service-account token capable of creating configs.
Supply the token securely via `DOPPLER_TOKEN`; do not attach tokens, Terraform
state, or full debug logs to an issue.

Use the minimal Terraform resource in the
[plain Terraform reproduction](live-reconciliation.md#plain-terraform-reproduction), then:

1. Select an unused branch name in a known existing test environment.
2. Call `GET /v3/configs/config?project=TEST_PROJECT&config=UNUSED_NAME`.
3. Try `terraform import doppler_config.probe TEST_PROJECT.ci.UNUSED_NAME`
   against the missing resource (with the resource declaration using that name).
4. Record the HTTP status/error and whether Terraform classifies it as missing.
5. From empty resource state, apply that exact config using the same token.
   Confirm the plan has one create and no updates/deletes before applying.

Observed initially: step 2 returned HTTP 400 with an access error, step 3 failed
refresh with that error, while step 5 successfully created the branch. A later
read of another known-missing branch with the same installed token returned 404,
so the original error is **not currently a consistent reproduction**.

Questions for Doppler maintainers:

- Under which permission/API conditions does a missing config return 400 versus
  404? Can callers with sufficient environment access get an unambiguous absence
  response while genuinely unauthorized reads remain errors?
- Is there an authoritative existence endpoint or list permission that guarantees
  completeness? If so, what pagination/filtering contract should clients use?
- Should Terraform's read handler change, or must this distinction be made by
  the API? It should not clear state solely on the ambiguous access-error text.

This is a prepared draft, not a filed upstream issue. Further evidence or an API
contract is needed before implementing a fallback.

## Operator guidance

On the ambiguous error, first verify permissions and remote identity through an
independently authorized view. If the config exists, restore access rather than
resetting its identity. If deletion is independently confirmed and recreation is
intended, deliberately migrate back to the new-config path after reviewing its
state and dependents. Do not automatically clear external-name annotations or
Terraform state, and do not delete existing configs just to make a test pass.
