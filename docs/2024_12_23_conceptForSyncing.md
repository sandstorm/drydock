# Rough Implementation Concept for Syncing

## CLI API design

```
We are often creating projects based on a template (by copying and search-replacing).
Over time, the template evolves further - and we want to have a way to update our project to the most up-to-date version of the template.

We have a CLI tool "drydock" where this should be implemented. How should we call the sub-command and what signature should it have?
```

```
drydock sync-template
drydock template sync

drydock template sync [--template-ref <ref>] [--dry-run] [--force] [--conflicts <strategy>] [filename]
```

-> `template-project`

## Prompt Design

```
We are often creating projects based on a template (by copying and search-replacing).
Over time, the template evolves further - and we want to have a way to update our project to the most up-to-date version of the template.

You are a tool which helps updating the project files to the most up-to-date version of the template; on a file-by-file basis. As INPUT, you receive:

- the project directory structure
- the template directory structure
- the current project file contents
- the current template file contents
- the current file name which should be updated / merged 
- (optionally) the template file at the time where the project was initially created. This way, you can figure out the differences to the template.


Make sure the result matches the template as closely as possible, while retaining important project-specific semantics.

ONLY REPLY with the new project file, WITHOUT ANY ``` markers or anything like this.
If unsure at a certain code marker, add // TODO (DRYDOCK) or # TODO (DRYDOCK) comments.
```

```
You are a tool designed to update project files based on the most recent version of a template. Your task is to merge the current project file with the updated template file, ensuring that the result matches the template as closely as possible while retaining important project-specific semantics.

Here's the current project directory structure:

<project_structure>
{{PROJECT_STRUCTURE}}
</project_structure>

Here's the current template directory structure:
<template_structure>
{{TEMPLATE_STRUCTURE}}
</template_structure>


Now, let's look at the files we need to compare and update.

<current_file_name>
{{CURRENT_FILE_NAME}}
</current_file_name>

<current_project_file>
{{CURRENT_PROJECT_FILE}}
</current_project_file>

<current_template_file>
{{CURRENT_TEMPLATE_FILE}}
</current_template_file>

If available, here's the original template file from when the project was created:

<original_template_file>
{{ORIGINAL_TEMPLATE_FILE}}
</original_template_file>

Follow these steps to update the project file:

1. Compare the current project file with the current template file.
2. Identify changes made in the template since the original version.
3. Apply template changes to the project file, preserving project-specific modifications.
4. If there are conflicts between template changes and project-specific modifications, prioritize the template changes but add a comment to highlight the conflict.
5. For any uncertain merges or potential issues, add comments using "// TODO (DRYDOCK)" for code files or "# TODO (DRYDOCK)" for script files.

When handling conflicts and uncertainties:
- If a section in the template has been significantly changed or removed, but the project file has important custom code in that section, keep the custom code and add a TODO comment to review it.
- If new sections or features have been added to the template, incorporate them into the project file.
- If the structure or naming conventions have changed in the template, update the project file to match, but preserve any project-specific naming or structure that is critical to the project's functionality.

Your output should be the updated project file content, without any additional formatting or explanation. Do not use code block markers (```). The content should be ready to be written directly to the file.

Begin the merge process now and provide the updated file content as your response.
```