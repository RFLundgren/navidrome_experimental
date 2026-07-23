import { withWidth } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import {
  useShowController,
  ShowContextProvider,
  useRecordContext,
  useShowContext,
  ReferenceManyField,
  Title as RaTitle,
} from 'react-admin'
import { useDispatch } from 'react-redux'
import AlbumGridView from '../album/AlbumGridView'
import { FilterableSongSection, Title, useResourceRefresh } from '../common'
import { setTrack } from '../actions'
import GenreActions from './GenreActions'

const useStyles = makeStyles(
  (theme) => ({
    actionsContainer: {
      paddingLeft: '.75rem',
      [theme.breakpoints.down('xs')]: {
        padding: '.5rem',
      },
    },
  }),
  {
    name: 'NDGenreShow',
  },
)

const GenreSongsSection = ({ record, titleKey, sort, dispatch }) => {
  const handleRowClick = (id, basePath, songRecord) => {
    dispatch(setTrack(songRecord))
    return false
  }
  if (!record) return null
  return (
    <FilterableSongSection
      target="genre_id"
      baseFilter={{ genre_id: record.id, missing: false }}
      sort={sort}
      titleKey={titleKey}
      rowClick={handleRowClick}
    />
  )
}

const GenreShowLayout = ({ width }) => {
  const showContext = useShowContext()
  const record = useRecordContext()
  const classes = useStyles()
  const dispatch = useDispatch()
  useResourceRefresh('genre', 'album', 'song')

  if (!record) return null

  return (
    <>
      <RaTitle title={<Title subTitle={record.name} />} />
      <div className={classes.actionsContainer}>
        <GenreActions record={record} />
      </div>
      <ReferenceManyField
        {...showContext}
        addLabel={false}
        reference="album"
        target="genre_id"
        sort={{ field: 'max_year', order: 'DESC' }}
        filter={{ genre_id: record.id }}
        perPage={0}
      >
        <AlbumGridView width={width} />
      </ReferenceManyField>
      <GenreSongsSection
        record={record}
        titleKey="resources.genre.topSongs"
        sort={{ field: 'play_count', order: 'DESC' }}
        dispatch={dispatch}
      />
      <GenreSongsSection
        record={record}
        titleKey="resources.genre.recentlyAdded"
        sort={{ field: 'recently_added', order: 'DESC' }}
        dispatch={dispatch}
      />
    </>
  )
}

const GenreShow = withWidth()((props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <GenreShowLayout {...controllerProps} width={props.width} />
    </ShowContextProvider>
  )
})

export default GenreShow
