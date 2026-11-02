try {
  const preference = localStorage.getItem('teststudio.theme') || 'system'
  const theme = preference === 'dark' || (preference !== 'light' && matchMedia('(prefers-color-scheme: dark)').matches) ? 'dark' : 'light'
  document.documentElement.dataset.theme = theme
  document.documentElement.style.colorScheme = theme
} catch {
  document.documentElement.dataset.theme = matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}
