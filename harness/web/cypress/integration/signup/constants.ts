export const CreateUserResponse = {
  access_token: 'token',
  token: {
    principal_id: 1,
    type: 'session',
    identifier: 'register',
    expires_at: 1715463431214,
    issued_at: 1712871431214,
    created_by: 15,
    uid: 'register'
  }
}

export const GetUserResponse = {
  uid: 'testuser',
  email: 'test@harness.io',
  display_name: 'testuser',
  admin: false,
  blocked: false,
  created: 1712871431212,
  updated: 1712871431212
}

export const signupPostCall = `/api/v1/register?include_cookie=true`
export const userGetCall = `/api/v1/user`
export const membershipGetCall = '/api/v1/user/memberships'
export const membershipQueryGetCall = '/api/v1/user/memberships?query='
