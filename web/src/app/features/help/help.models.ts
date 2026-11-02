export interface HelpPage {
  id: string
  title: string
  introduction: string[]
  concepts: { title: string; description: string }[]
  tutorials: { title: string; steps: string[] }[]
}
