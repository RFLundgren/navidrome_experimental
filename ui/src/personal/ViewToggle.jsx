import PropTypes from 'prop-types'
import { useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { setViewToggle } from '../actions'
import { FormControl, FormControlLabel, Switch } from '@material-ui/core'

// Generic personal-menu visibility toggle, for switches that are all
// identical in behavior (show/hide one sidebar entry) and don't warrant
// their own dedicated component/action - see FolderViewToggle.jsx for the
// original single-purpose version this generalizes.
export const ViewToggle = ({ settingsKey, labelKey }) => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const enabled = useSelector((state) => state.settings[settingsKey])

  const toggle = (event) => {
    dispatch(setViewToggle(settingsKey, event.target.checked))
  }

  return (
    <FormControl>
      <FormControlLabel
        control={
          <Switch
            id={settingsKey}
            color="primary"
            checked={enabled !== false}
            onChange={toggle}
          />
        }
        label={<span>{translate(labelKey)}</span>}
      />
    </FormControl>
  )
}

ViewToggle.propTypes = {
  settingsKey: PropTypes.string.isRequired,
  labelKey: PropTypes.string.isRequired,
}
