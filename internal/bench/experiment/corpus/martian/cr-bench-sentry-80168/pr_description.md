## feat(workflow_engine): Add in hook for producing occurrences from the stateful detector

This adds a hook that can be implemented to produce an occurrence specific to the detector that is subclassing the StatefulDetector.

Also change the signature of evaluate to return a dict keyed by groupkey instead of a list. This helps avoid the chance of duplicate results for the same group key.

<!-- Describe your PR here. -->