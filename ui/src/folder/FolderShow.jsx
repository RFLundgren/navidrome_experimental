import React, { useEffect, useState } from 'react'
import {
  ReferenceManyField,
  ShowContextProvider,
  useShowContext,
  useShowController,
  Title as RaTitle,
  Datagrid,
  FunctionField,
  SimpleShowLayout,
  useTranslate,
  Pagination,
  useListContext,
  NumberField,
  Filter,
  SearchInput,
  ListToolbar,
  linkToRecord,
} from 'react-admin'
import { useSelector } from 'react-redux'
import { Link } from 'react-router-dom'
import FolderIcon from '@material-ui/icons/Folder'
import {
  makeStyles,
  Typography,
  Box,
  Card,
  CardMedia,
  CardContent,
} from '@material-ui/core'
import Breadcrumbs from './Breadcrumbs'
import FolderSongs from './FolderSongs'
import FolderListActions from './FolderListActions'
import FolderGridView from './FolderGridView'
import subsonic from '../subsonic'
import config from '../config'
import { SafeHTML } from '../common/SafeHTML'
import {
  useResourceRefresh,
  Title,
  FolderContextMenu,
  DurationField,
  SizeField,
} from '../common'

const useStyles = makeStyles({
  icon: {
    verticalAlign: 'middle',
    marginRight: '10px',
  },
  sectionTitle: {
    marginTop: '20px',
    marginBottom: '10px',
    fontWeight: 'bold',
  },
  row: {
    '&:hover': {
      '& $contextMenu': {
        visibility: 'visible',
      },
    },
  },
  contextMenu: {
    visibility: 'hidden',
  },
  toolbar: {
    justifyContent: 'flex-start',
  },
  artistCard: {
    display: 'flex',
    marginTop: '16px',
    marginBottom: '16px',
    padding: '16px',
    alignItems: 'flex-start',
  },
  artistImage: {
    width: '144px',
    height: '144px',
    minWidth: '144px',
    borderRadius: '72px',
    objectFit: 'cover',
  },
  artistContent: {
    paddingTop: 0,
    paddingBottom: '0 !important',
    minWidth: 0,
    flex: 1,
  },
  artistLink: {
    color: 'inherit',
    textDecoration: 'none',
    '&:hover': {
      textDecoration: 'underline',
    },
  },
  artistBiography: {
    marginTop: '8px',
    wordBreak: 'break-word',
  },
  '@media (max-width: 600px)': {
    artistCard: {
      flexDirection: 'column',
    },
    artistImage: {
      width: '112px',
      height: '112px',
      minWidth: '112px',
      borderRadius: '56px',
      marginBottom: '12px',
    },
    artistContent: {
      paddingLeft: 0,
    },
  },
})

const FolderFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="name" alwaysOn />
  </Filter>
)

const FolderArtistInfo = () => {
  const classes = useStyles()
  const { record } = useShowContext()
  const [artistInfo, setArtistInfo] = useState(null)
  const artistId = record?.artistId

  useEffect(() => {
    let cancelled = false
    setArtistInfo(null)

    if (!artistId) {
      return () => {
        cancelled = true
      }
    }

    subsonic
      .getArtistInfo(artistId)
      .then((resp) => {
        if (cancelled) return
        const data = resp.json['subsonic-response']
        if (data?.status === 'ok') {
          setArtistInfo(data.artistInfo || null)
        }
      })
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.error('error loading artist info for folder', e)
      })

    return () => {
      cancelled = true
    }
  }, [artistId])

  if (!artistId) return null

  const artist = {
    id: artistId,
    name: record.artistName || record.name,
  }
  const biography = artistInfo?.biography

  return (
    <Card className={classes.artistCard} elevation={1}>
      <CardMedia
        component="img"
        src={subsonic.getCoverArtUrl(artist, config.uiCoverArtSize)}
        title={artist.name}
        className={classes.artistImage}
      />
      <CardContent className={classes.artistContent}>
        <Typography variant="h5">
          <Link
            to={linkToRecord('/artist', artist.id, 'show')}
            className={classes.artistLink}
          >
            {artist.name}
          </Link>
        </Typography>
        {biography && (
          <Typography variant="body1" className={classes.artistBiography}>
            <SafeHTML>{biography}</SafeHTML>
          </Typography>
        )}
      </CardContent>
    </Card>
  )
}

const SubfoldersSection = (props) => {
  const { ids = [], loaded, loading, filterValues = {} } = useListContext()
  const classes = useStyles()
  const translate = useTranslate()
  const folderView = useSelector((state) => state.folderView)

  // React-admin may not always expose `total` here, but an empty `ids` list is
  // authoritative. Returning null prevents the iterator from rendering its
  // generic "No results found" placeholder for folders without subfolders.
  if (loaded && ids.length === 0 && !filterValues.name) return null

  return (
    <>
      <Box className={classes.sectionTitle}>
        <Typography variant="h6">
          {translate('resources.folder.fields.subfolders')}
        </Typography>
      </Box>
      <ListToolbar
        classes={{ toolbar: classes.toolbar }}
        filters={<FolderFilter />}
        actions={null}
        {...props}
      />
      {folderView.grid ? (
        <FolderGridView {...props} />
      ) : (
        <Datagrid rowClick="show" classes={{ row: classes.row }}>
          <FunctionField
            source="name"
            render={(record) => (
              <>
                <FolderIcon className={classes.icon} />
                {record.name}
              </>
            )}
          />
          <FolderContextMenu
            source="name"
            className={classes.contextMenu}
            showLove={false}
          />
        </Datagrid>
      )}
    </>
  )
}

const SongsSection = ({ record, ...props }) => {
  const { total, loaded, filterValues } = useListContext()
  const classes = useStyles()
  const translate = useTranslate()

  if (loaded && total === 0 && !filterValues.title) return null

  return (
    <>
      <Box className={classes.sectionTitle}>
        <Typography variant="h6">
          {translate('resources.folder.fields.songs')}
        </Typography>
      </Box>
      <FolderSongs folder={record} {...props} />
    </>
  )
}

const FolderShowLayout = (props) => {
  const { record, loading } = useShowContext(props)
  useResourceRefresh('folder', 'song')

  if (loading || !record) return null

  const displayName =
    (record.path === '' || record.path === '.') && record.breadcrumbs?.length
      ? record.breadcrumbs[0].name
      : record.name

  return (
    <>
      <RaTitle title={<Title subTitle={displayName} />} />
      <FolderListActions {...props} />
      <SimpleShowLayout record={record}>
        <FolderHeader />
        <FolderArtistInfo />

        <ReferenceManyField
          reference="folder"
          target="parent_id"
          label=""
          sort={{ field: 'name', order: 'ASC' }}
          perPage={500}
          pagination={<Pagination rowsPerPageOptions={[100, 250, 500, 1000]} />}
          fullWidth
        >
          <SubfoldersSection {...props} />
        </ReferenceManyField>

        <ReferenceManyField
          reference="song"
          target="folder_id"
          label=""
          sort={{ field: 'album', order: 'ASC' }}
          perPage={500}
          pagination={<Pagination rowsPerPageOptions={[100, 250, 500, 1000]} />}
          fullWidth
        >
          <SongsSection record={record} />
        </ReferenceManyField>
      </SimpleShowLayout>
    </>
  )
}

const FolderShow = (props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <FolderShowLayout {...props} {...controllerProps} />
    </ShowContextProvider>
  )
}

const FolderHeader = () => {
  const { record } = useShowContext()
  if (!record || !record.breadcrumbs) return null
  return <Breadcrumbs breadcrumbs={record.breadcrumbs} />
}

export default FolderShow
