import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import { TAG_DASHBOARDS } from './tagDashboard/tagDashboards'
import { TagDashboardList } from './tagDashboard/TagDashboardList'
import { TagDashboardShow } from './tagDashboard/TagDashboardShow'

const tagDashboardRoutes = Object.values(TAG_DASHBOARDS).flatMap((dashboard) => [
  <Route
    exact
    path={dashboard.path}
    render={() => <TagDashboardList dashboard={dashboard} />}
    key={`${dashboard.key}-list`}
  />,
  <Route
    exact
    path={`${dashboard.path}/:tag`}
    render={({ match }) => (
      <TagDashboardShow dashboard={dashboard} match={match} />
    )}
    key={`${dashboard.key}-show`}
  />,
])

const routes = [
  <Route exact path="/personal" render={() => <Personal />} key={'personal'} />,
  ...tagDashboardRoutes,
]

export default routes
