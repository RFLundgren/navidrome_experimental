import React, { useEffect, useState } from 'react'
import PropTypes from 'prop-types'
import { useHistory } from 'react-router-dom'
import { Loading, useTranslate } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { Title } from '../common'
import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'
import { genreGradient } from '../genre/genreColor'

const useStyles = makeStyles((theme) => ({
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
    gap: '1rem',
    padding: '0.5rem 1rem 1rem',
  },
  chip: {
    borderRadius: '12px',
    padding: '1.25rem',
    color: '#fff',
    cursor: 'pointer',
    minHeight: '96px',
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'space-between',
    boxShadow: '0 2px 6px rgba(0,0,0,0.25)',
    transition: 'transform 150ms ease, box-shadow 150ms ease',
    outline: 'none',
    '&:hover, &:focus-visible': {
      transform: 'translateY(-2px)',
      boxShadow: '0 4px 12px rgba(0,0,0,0.35)',
    },
  },
  name: {
    fontSize: '1.15rem',
    fontWeight: 600,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    textShadow: '0 1px 2px rgba(0,0,0,0.35)',
  },
  counts: {
    fontSize: '0.85rem',
    opacity: 0.92,
    textShadow: '0 1px 2px rgba(0,0,0,0.35)',
  },
  empty: {
    padding: '2rem',
    textAlign: 'center',
  },
}))

// Fetches /mediaFileTag/counts for the dashboard's source, narrows to its
// prefix if any (splitting AI Auto-Tagging's combined genre:/mood: tag
// namespace into two separate dashboards), and renders a chip per distinct
// value - visually the same chip grid Genre Exploration uses, but backed by
// a plain fetch instead of a react-admin Resource, since tag values have no
// backing DB table/id of their own.
export const TagDashboardList = ({ dashboard }) => {
  const classes = useStyles()
  const history = useHistory()
  const translate = useTranslate()
  const [tags, setTags] = useState(null)
  const [error, setError] = useState(false)

  useEffect(() => {
    let cancelled = false
    setTags(null)
    setError(false)
    httpClient(`${REST_URL}/mediaFileTag/counts?source=${dashboard.source}`)
      .then((res) => {
        if (cancelled) return
        const all = res.json || []
        const filtered = dashboard.prefix
          ? all.filter((t) => t.tagName.startsWith(dashboard.prefix))
          : all
        const mapped = filtered
          .map((t) => ({
            fullName: t.tagName,
            name: dashboard.prefix
              ? t.tagName.slice(dashboard.prefix.length)
              : t.tagName,
            count: t.count,
          }))
          .sort((a, b) => a.name.localeCompare(b.name))
        setTags(mapped)
      })
      .catch(() => !cancelled && setError(true))
    return () => {
      cancelled = true
    }
  }, [dashboard])

  const goToTag = (fullName) =>
    history.push(`${dashboard.path}/${encodeURIComponent(fullName)}`)

  const title = translate(`resources.${dashboard.resourceName}.name`, {
    smart_count: 2,
  })

  if (error) {
    return (
      <>
        <Title title={title} />
        <div className={classes.empty}>{translate('ra.page.error')}</div>
      </>
    )
  }

  if (!tags) return <Loading />

  return (
    <>
      <Title title={title} />
      {tags.length === 0 ? (
        <div className={classes.empty}>
          {translate(`resources.${dashboard.resourceName}.empty`)}
        </div>
      ) : (
        <div className={classes.grid}>
          {tags.map((tag) => (
            <div
              key={tag.fullName}
              className={classes.chip}
              style={{ background: genreGradient(tag.name) }}
              onClick={() => goToTag(tag.fullName)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') goToTag(tag.fullName)
              }}
              role="button"
              tabIndex={0}
            >
              <div className={classes.name}>{tag.name}</div>
              <div className={classes.counts}>
                {translate('resources.tagDashboard.chipSongCount', {
                  smart_count: tag.count,
                })}
              </div>
            </div>
          ))}
        </div>
      )}
    </>
  )
}

TagDashboardList.propTypes = {
  dashboard: PropTypes.shape({
    path: PropTypes.string.isRequired,
    source: PropTypes.string.isRequired,
    prefix: PropTypes.string,
    resourceName: PropTypes.string.isRequired,
  }).isRequired,
}
