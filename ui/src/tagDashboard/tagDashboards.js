// Config for the three tag-based dashboards (AI Genre, AI Mood, My Tags).
// Each is a lighter custom page rather than a full react-admin Resource,
// since tag values have no backing DB table/id the way genre does - they're
// derived aggregates over media_file_tag. `prefix` splits AI Auto-Tagging's
// combined "genre:"/"mood:" tag namespace into two separate dashboards;
// My Tags has no prefix since personal tags aren't categorized that way.
export const TAG_DASHBOARDS = {
  aiGenre: {
    key: 'aiGenre',
    path: '/aiGenreTags',
    source: 'ai',
    prefix: 'genre:',
    resourceName: 'aiGenre',
    settingsFlag: 'showAiGenreView',
  },
  aiMood: {
    key: 'aiMood',
    path: '/aiMoodTags',
    source: 'ai',
    prefix: 'mood:',
    resourceName: 'aiMood',
    settingsFlag: 'showAiMoodView',
  },
  myTags: {
    key: 'myTags',
    path: '/myTags',
    source: 'user',
    prefix: '',
    resourceName: 'myTags',
    settingsFlag: 'showMyTagsView',
  },
}
