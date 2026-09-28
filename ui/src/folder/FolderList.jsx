import React, { useEffect, useState } from 'react'
import {
  List,
  Datagrid,
  useTranslate,
  FunctionField,
  useDataProvider,
  Pagination,
} from 'react-admin'
import FolderIcon from '@material-ui/icons/Folder'
import { makeStyles } from '@material-ui/core'
import { useSelector } from 'react-redux'
import { Title, FolderContextMenu } from '../common'
import FolderListActions from './FolderListActions'
import FolderGridView from './FolderGridView'

const useStyles = makeStyles({
  icon: {
    verticalAlign: 'middle',
    marginRight: '10px',
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
})

const getLibraryName = (record, librariesById) =>
  librariesById[String(record?.libraryId)] || record?.name

const LibraryRootGridView = ({ data, librariesById, ...props }) => {
  if (!data) return <FolderGridView data={data} {...props} />

  const renamedData = Object.fromEntries(
    Object.entries(data).map(([id, record]) => [
      id,
      {
        ...record,
        name: getLibraryName(record, librariesById),
      },
    ]),
  )

  return <FolderGridView {...props} data={renamedData} />
}

const FolderList = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const folderView = useSelector((state) => state.folderView)
  const { userLibraries = [], selectedLibraries = [] } = useSelector(
    (state) => state.library || {},
  )
  const [libraries, setLibraries] = useState([])
  const [rootFolderId, setRootFolderId] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setRootFolderId(null)

    // The /api/library resource is admin-only. The current user's accessible
    // libraries are already loaded by LibrarySelector from the user's own
    // record and stored in Redux, so use that source here as well. This keeps
    // the folder browser working for normal users instead of turning a 403 into
    // an empty "No libraries found" view.
    const selectedIds = new Set(selectedLibraries.map(String))
    const activeLibraries = selectedIds.size
      ? userLibraries.filter((library) =>
          selectedIds.has(String(library.id)),
        )
      : userLibraries

    setLibraries(activeLibraries)

    // With more than one active library, the folder root becomes a list of
    // library roots. A single library keeps the existing direct-to-content
    // behaviour and avoids adding an unnecessary navigation level.
    if (activeLibraries.length !== 1) {
      setLoading(false)
      return () => {
        cancelled = true
      }
    }

    const libraryId = String(activeLibraries[0].id)
    dataProvider
      .getList('folder', {
        pagination: { page: 1, perPage: 1 },
        sort: { field: 'id', order: 'ASC' },
        filter: { library_id: libraryId, parent_id: '' },
      })
      .then(({ data: rootData }) => {
        if (cancelled) return
        if (rootData.length > 0) {
          setRootFolderId(rootData[0].id)
        }
        setLoading(false)
      })
      .catch(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [dataProvider, selectedLibraries, userLibraries])

  if (loading) return null
  if (libraries.length === 0)
    return (
      <div style={{ padding: '20px' }}>
        No libraries found. Please scan your music first.
      </div>
    )

  const multipleLibraries = libraries.length > 1
  if (!multipleLibraries && !rootFolderId)
    return (
      <div style={{ padding: '20px' }}>
        No library root folder found. Please rescan your music.
      </div>
    )

  const librariesById = Object.fromEntries(
    libraries.map((library) => [String(library.id), library.name]),
  )
  const filter = multipleLibraries
    ? {
        parent_id: '',
        library_id: libraries.map((library) => String(library.id)),
      }
    : { parent_id: rootFolderId }

  return (
    <List
      {...props}
      perPage={500}
      sort={
        multipleLibraries
          ? { field: 'library_name', order: 'ASC' }
          : { field: 'name', order: 'ASC' }
      }
      filter={filter}
      actions={<FolderListActions />}
      pagination={<Pagination rowsPerPageOptions={[100, 250, 500, 1000]} />}
      title={<Title title={translate('menu.folders')} />}
    >
      {folderView.grid ? (
        multipleLibraries ? (
          <LibraryRootGridView librariesById={librariesById} {...props} />
        ) : (
          <FolderGridView {...props} />
        )
      ) : (
        <Datagrid rowClick="show" classes={{ row: classes.row }}>
          <FunctionField
            source="name"
            render={(record) => {
              if (!record) return null
              const name = multipleLibraries
                ? getLibraryName(record, librariesById)
                : record.name
              if (!name) return null
              return (
                <>
                  <FolderIcon className={classes.icon} />
                  {name}
                </>
              )
            }}
          />
          {multipleLibraries ? (
            <FunctionField
              render={(record) => (
                <FolderContextMenu
                  record={{
                    ...record,
                    name: getLibraryName(record, librariesById),
                  }}
                  className={classes.contextMenu}
                  showLove={false}
                />
              )}
            />
          ) : (
            <FolderContextMenu
              source="name"
              className={classes.contextMenu}
              showLove={false}
            />
          )}
        </Datagrid>
      )}
    </List>
  )
}

export default FolderList
