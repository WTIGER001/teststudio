export interface HelpPage {
  id: string
  title: string
  introduction: string[]
  concepts: { title: string; description: string }[]
  tutorials: { title: string; steps: string[]; examples?: import('./help-example.component').HelpExample[]; highlights?: { anchor: string; label: string }[] }[]
}
