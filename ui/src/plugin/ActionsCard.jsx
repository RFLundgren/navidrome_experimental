import React, { useState, useCallback } from 'react'
import PropTypes from 'prop-types'
import { Card, CardContent, Typography, Box, Button } from '@material-ui/core'
import CircularProgress from '@material-ui/core/CircularProgress'
import Alert from '@material-ui/lab/Alert'
import { useTranslate } from 'react-admin'
import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'

// One button per action declared in the plugin's manifest.json "actions"
// array (e.g. a "Test Connection" button) - each POSTs to
// /plugin/:id/actions/:name and shows the result inline, success or error.
export const ActionsCard = ({ manifest, pluginId, enabled, classes }) => {
  const translate = useTranslate()
  const [running, setRunning] = useState(null)
  const [results, setResults] = useState({})

  const runAction = useCallback(
    (name) => {
      setRunning(name)
      setResults((prev) => ({ ...prev, [name]: null }))
      httpClient(`${REST_URL}/plugin/${pluginId}/actions/${name}`, {
        method: 'POST',
      })
        .then((res) => {
          setResults((prev) => ({
            ...prev,
            [name]: { severity: 'success', message: res.json?.message },
          }))
        })
        .catch((err) => {
          setResults((prev) => ({
            ...prev,
            [name]: { severity: 'error', message: err?.message },
          }))
        })
        .finally(() => setRunning(null))
    },
    [pluginId],
  )

  const actions = manifest?.actions
  if (!actions || actions.length === 0) {
    return null
  }

  return (
    <Card className={classes.section}>
      <CardContent>
        <Typography variant="h6" className={classes.sectionTitle}>
          {translate('resources.plugin.sections.actions')}
        </Typography>

        {!enabled && (
          <Box mb={2}>
            <Alert severity="info">
              {translate('resources.plugin.messages.actionsDisabledHelp')}
            </Alert>
          </Box>
        )}

        {actions.map((action) => {
          const result = results[action.name]
          const isRunning = running === action.name
          return (
            <Box key={action.name} mb={2}>
              <Box display="flex" alignItems="center">
                <Button
                  variant="outlined"
                  color="primary"
                  disabled={!enabled || running !== null}
                  onClick={() => runAction(action.name)}
                >
                  {action.label}
                </Button>
                {isRunning && (
                  <Box ml={2} display="flex" alignItems="center">
                    <CircularProgress size={16} />
                    <Box ml={1}>
                      <Typography variant="body2">
                        {translate('resources.plugin.messages.actionRunning')}
                      </Typography>
                    </Box>
                  </Box>
                )}
              </Box>
              {action.description && (
                <Typography variant="body2" color="textSecondary">
                  {action.description}
                </Typography>
              )}
              {result && (
                <Box mt={1}>
                  <Alert severity={result.severity}>{result.message}</Alert>
                </Box>
              )}
            </Box>
          )
        })}
      </CardContent>
    </Card>
  )
}

ActionsCard.propTypes = {
  manifest: PropTypes.shape({
    actions: PropTypes.arrayOf(
      PropTypes.shape({
        name: PropTypes.string.isRequired,
        label: PropTypes.string.isRequired,
        description: PropTypes.string,
      }),
    ),
  }),
  pluginId: PropTypes.string.isRequired,
  enabled: PropTypes.bool,
  classes: PropTypes.object.isRequired,
}
