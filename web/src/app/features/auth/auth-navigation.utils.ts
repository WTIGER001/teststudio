export function authenticationReturnPath(route: string): string {
  const destination = new URL(route, 'http://teststudio.local')
  destination.searchParams.delete('signIn'); destination.searchParams.delete('providerLogout')
  return destination.pathname + destination.search + destination.hash
}

export function providerLoginPath(provider: string, route: string): string {
  return `/auth/login/${encodeURIComponent(provider)}?${new URLSearchParams({ returnTo: authenticationReturnPath(route) })}`
}

export function authenticationFeedback(route: string): { message: string; error: boolean } {
  const query = new URL(route, 'http://teststudio.local').searchParams
  switch (query.get('signIn')) {
    case 'cancelled': return { message: 'Sign-in was cancelled or denied. Choose a provider to try again.', error: false }
    case 'expired': return { message: 'This sign-in request expired or was already used. Start a new sign-in.', error: true }
    case 'unavailable': return { message: 'The selected sign-in provider is unavailable. Try again or choose another sign-in method.', error: true }
    case 'failed': return { message: 'Sign-in could not be verified. Start again with the correct provider and account.', error: true }
  }
  switch (query.get('providerLogout')) {
    case 'complete': return { message: 'Provider sign-out completed. The TestStudio session that started it remains closed.', error: false }
    case 'unconfirmed': return { message: 'Provider sign-out could not be confirmed. The TestStudio session that started it remains closed.', error: false }
  }
  return { message: '', error: false }
}
