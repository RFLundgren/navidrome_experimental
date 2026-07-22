import React, { useEffect, useState } from 'react'
import PropTypes from 'prop-types'
import { TextField as MuiTextField } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import {
  ReferenceManyField,
  Datagrid,
  TextField,
  useTranslate,
} from 'react-admin'
import { DurationField } from './DurationField'
import { Pagination } from './Pagination'

const useStyles = makeStyles(
  (theme) => ({
    section: {
      margin: '1rem 1.5rem',
    },
    header: {
      display: 'flex',
      alignItems: 'baseline',
      justifyContent: 'space-between',
      flexWrap: 'wrap',
      marginBottom: '0.5rem',
    },
    sectionTitle: {
      margin: 0,
    },
    search: {
      minWidth: '12rem',
      [theme.breakpoints.down('xs')]: {
        width: '100%',
      },
    },
  }),
  { name: 'NDFilterableSongSection' },
)

// Shared song list used by the Genre, AI Genre, AI Mood and My Tags show
// pages: a ReferenceManyField with real pagination (instead of the previous
// pagination={null}, which silently hid every song past the first page) and
// a search box scoped to source/artist/album via the same "title" filter
// SongList.jsx's search uses, which already does a broad text match rather
// than a literal title-only one. The search text is debounced into a
// separate "applied" value, which is also used as the field's key - forcing
// react-admin to remount (and reset back to page 1) whenever a new search is
// applied, so the user never lands on a now-empty later page.
export const FilterableSongSection = ({
  reference,
  target,
  baseFilter,
  sort,
  perPage,
  titleKey,
  rowClick,
}) => {
  const translate = useTranslate()
  const classes = useStyles()
  const [searchText, setSearchText] = useState('')
  const [appliedSearch, setAppliedSearch] = useState('')

  useEffect(() => {
    const handle = setTimeout(() => setAppliedSearch(searchText.trim()), 400)
    return () => clearTimeout(handle)
  }, [searchText])

  const filter = appliedSearch
    ? { ...baseFilter, title: appliedSearch }
    : baseFilter

  return (
    <div className={classes.section}>
      <div className={classes.header}>
        {titleKey && (
          <h6 className={classes.sectionTitle}>{translate(titleKey)}</h6>
        )}
        <MuiTextField
          variant="outlined"
          size="small"
          margin="dense"
          placeholder={translate('ra.action.search')}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          className={classes.search}
        />
      </div>
      <ReferenceManyField
        key={appliedSearch}
        reference={reference}
        target={target}
        filter={filter}
        sort={sort}
        perPage={perPage}
        pagination={<Pagination />}
      >
        <Datagrid rowClick={rowClick} bulkActionButtons={false}>
          <TextField source="title" />
          <TextField source="artist" />
          <TextField source="album" />
          <DurationField source="duration" />
        </Datagrid>
      </ReferenceManyField>
    </div>
  )
}

FilterableSongSection.propTypes = {
  reference: PropTypes.string,
  target: PropTypes.string.isRequired,
  baseFilter: PropTypes.object.isRequired,
  sort: PropTypes.object.isRequired,
  perPage: PropTypes.number,
  titleKey: PropTypes.string,
  rowClick: PropTypes.func.isRequired,
}

FilterableSongSection.defaultProps = {
  reference: 'song',
  perPage: 25,
}
