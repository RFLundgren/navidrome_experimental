import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FolderList from './FolderList'

const mockDataProvider = {
  getList: vi.fn(),
}

let mockState

vi.mock('react-redux', () => ({
  useSelector: (selector) => selector(mockState),
}))

vi.mock('react-admin', () => ({
  List: ({ children, filter, sort }) => (
    <div
      data-testid="folder-list"
      data-filter={JSON.stringify(filter)}
      data-sort={JSON.stringify(sort)}
    >
      {children}
    </div>
  ),
  Datagrid: ({ children }) => <div>{children}</div>,
  FunctionField: () => null,
  useTranslate: () => (key) => key,
  useDataProvider: () => mockDataProvider,
  Pagination: () => null,
}))

vi.mock('@material-ui/core', () => ({
  makeStyles: () => () => ({
    icon: 'icon',
    row: 'row',
    contextMenu: 'contextMenu',
  }),
}))

vi.mock('@material-ui/icons/Folder', () => ({
  default: () => null,
}))

vi.mock('../common', () => ({
  Title: () => null,
  FolderContextMenu: () => null,
}))

vi.mock('./FolderListActions', () => ({
  default: () => null,
}))

vi.mock('./FolderGridView', () => ({
  default: () => null,
}))

describe('FolderList library roots', () => {
  const libraries = [
    { id: 1, name: 'Music Library', path: '/music' },
    { id: 2, name: 'Hörbücher', path: '/hoerbucher' },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
    mockState = {
      folderView: { grid: false },
      library: { userLibraries: libraries, selectedLibraries: ['1', '2'] },
    }
  })

  it('uses all selected user libraries as physical root folders without querying the admin-only library API', async () => {
    render(<FolderList basePath="/folder" resource="folder" />)

    const list = await screen.findByTestId('folder-list')
    expect(JSON.parse(list.dataset.filter)).toEqual({
      parent_id: '',
      library_id: ['1', '2'],
    })
    expect(JSON.parse(list.dataset.sort)).toEqual({
      field: 'library_name',
      order: 'ASC',
    })
    expect(mockDataProvider.getList).not.toHaveBeenCalled()
  })

  it('keeps the direct folder view when only one library is selected', async () => {
    mockState.library.selectedLibraries = ['2']
    mockDataProvider.getList.mockImplementation((resource, params) => {
      if (resource === 'folder') {
        expect(params.filter).toEqual({ library_id: '2', parent_id: '' })
        return Promise.resolve({
          data: [{ id: 'root-hoerbuch', libraryId: 2, name: '.' }],
          total: 1,
        })
      }
      throw new Error(`Unexpected resource: ${resource}`)
    })

    render(<FolderList basePath="/folder" resource="folder" />)

    const list = await screen.findByTestId('folder-list')
    expect(JSON.parse(list.dataset.filter)).toEqual({
      parent_id: 'root-hoerbuch',
    })
    expect(JSON.parse(list.dataset.sort)).toEqual({
      field: 'name',
      order: 'ASC',
    })
    await waitFor(() => expect(mockDataProvider.getList).toHaveBeenCalledTimes(1))
  })

  it('treats an empty selection as all accessible libraries', async () => {
    mockState.library.selectedLibraries = []

    render(<FolderList basePath="/folder" resource="folder" />)

    const list = await screen.findByTestId('folder-list')
    expect(JSON.parse(list.dataset.filter)).toEqual({
      parent_id: '',
      library_id: ['1', '2'],
    })
    expect(mockDataProvider.getList).not.toHaveBeenCalled()
  })
})
