import { createFileRoute } from '@tanstack/react-router'
import HomePage from './page'

export const indexRoute = createFileRoute('/')({
  component: HomePage,
}) 
