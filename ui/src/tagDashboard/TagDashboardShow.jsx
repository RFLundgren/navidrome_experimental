import React from 'react'
import PropTypes from 'prop-types'
import {
  RecordContextProvider,
  ReferenceManyField,
  Datagrid,
  TextField,
  Title as RaTitle,
  TopToolbar,
  sanitizeListRestProps,
} from 'react-admin'
import { useDispatch } from 'react-redux'
import { makeStyles } from '@material-ui/core/styles'
import {
  DurationField,
  Title,
  ShuffleAllButton,
  useResourceRefresh,
} from '../common'
import { setTrack } from '../actions'
import { CreatePlaylistFromTagButton } from './CreatePlaylistFromTagButton'

const useStyles = makeStyles(
  (theme) => ({
    actionsContainer: {
      paddingLeft: '.75rem',
      [theme.breakpoints.down('xs')]: {
        padding: '.5rem',
      },
    },
    toolbar: {
      minHeight: 'auto',
      padding: '0 !important',
      background: 'transparent',
      boxShadow: 'none',
      '& .MuiToolbar-root': {
        minHeight: 'auto',
        padding: '0 !important',
        background: 'transparent',
      },
    },
    section: {
      margin: '1rem 1.5rem',
    },
  }),
  { name: 'NDTagDashboardShow' },
)

// Per-tag landing page for the AI Genre / AI Mood / My Tags dashboards.
// Unlike Genre's show page, there's no Albums/Top Songs split here - tags
// are per-song, not per-album (a single album can have songs carrying
// different tags), so this is a plain song list plus Create Playlist,
// wrapped in a RecordContextProvider (not a real react-admin Resource
// record, since tag values have no backing DB table) purely so
// ReferenceManyField has the context it expects.
export const TagDashboardShow = ({ dashboard, match }) => {
  const classes = useStyles()
  const dispatch = useDispatch()
  useResourceRefresh('song')

  const tagName = decodeURIComponent(match.params.tag)
  const displayName = dashboard.prefix
    ? tagName.slice(dashboard.prefix.length)
    : tagName

  const handleRowClick = (id, basePath, songRecord) => {
    dispatch(setTrack(songRecord))
    return false
  }

  return (
    <RecordContextProvider value={{ id: tagName, name: displayName }}>
      <RaTitle title={<Title subTitle={displayName} />} />
      <div className={classes.actionsContainer}>
        <TopToolbar className={classes.toolbar} {...sanitizeListRestProps({})}>
          <ShuffleAllButton filters={{ user_tag: tagName }} />
          <CreatePlaylistFromTagButton
            tagName={tagName}
            displayName={displayName}
            source={dashboard.source}
          />
        </TopToolbar>
      </div>
      <div className={classes.section}>
        <ReferenceManyField
          reference="song"
          target="user_tag"
          filter={{ user_tag: tagName, missing: false }}
          sort={{ field: 'recently_added', order: 'DESC' }}
          perPage={100}
          pagination={null}
        >
          <Datagrid rowClick={handleRowClick} bulkActionButtons={false}>
            <TextField source="title" />
            <TextField source="artist" />
            <TextField source="album" />
            <DurationField source="duration" />
          </Datagrid>
        </ReferenceManyField>
      </div>
    </RecordContextProvider>
  )
}

TagDashboardShow.propTypes = {
  dashboard: PropTypes.shape({
    prefix: PropTypes.string,
    source: PropTypes.string.isRequired,
  }).isRequired,
  match: PropTypes.shape({
    params: PropTypes.shape({
      tag: PropTypes.string.isRequired,
    }).isRequired,
  }).isRequired,
}
