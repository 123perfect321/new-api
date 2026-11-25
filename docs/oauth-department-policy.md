# OIDC Department Group and Initial Quota Policy

## Goal

After an OIDC/SSO login:

- derive the department from the identity claims;
- place the user in the matching New API group;
- create the group when it does not exist;
- apply a configurable initial quota for a new user;
- never fall back to `default` when a department claim is available.

No department names are hard-coded in the implementation.

## Department key

New API reads the department in this order:

1. `org_name`
2. `organization_name`
3. `department_name`
4. `org_id`
5. `organization_id`
6. `department_id`

The first non-empty value is used as the group name. If the identity provider only returns an organization ID, that ID is used as the group key.

## Quota configuration

Default quota for new users, expressed in display credits:

```text
OIDC_DEFAULT_INITIAL_QUOTA_CREDITS=800
```

Optional per-department overrides:

```text
OIDC_DEPARTMENT_QUOTAS_JSON={"org-id-1":800,"org-id-2":1200,"部门名称":600}
```

Keys may be:

- organization ID;
- organization name;
- normalized group name.

The value is a display-credit value. New API converts it to internal quota using `QuotaPerUnit`.

Example: with `QuotaPerUnit=500000`, a value of `800` produces `400000000` internal quota.

## Important behavior

- Existing users only get their group updated on subsequent SSO logins.
- Existing users do not receive the initial quota again.
- Invalid configuration is logged and the default quota is used.
- The fallback group is `default` only when the identity provider returns no usable department claim.
