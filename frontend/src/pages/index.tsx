import { createFileRoute } from '@tanstack/react-router'
import HomePage from './AboutPage'

export const indexRoute = createFileRoute('/')({
  component: HomePage,
}) 
