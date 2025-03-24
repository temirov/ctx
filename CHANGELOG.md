# Changelog

## [v0.0.2] - 2025-03-23

### What's New 🎉

1. **Using `.ignore` Instead of `.ignore`:**  
   The tool now looks for a file named **.ignore** for exclusion patterns.

2. **Processing `.gitignore` by Default:**  
   The tool loads **.gitignore** by default if present. Use the `--no-gitignore` flag to disable this.

3. **Folder Exclusion Flags:**  
   Both **-e** and **--e** flags now work for specifying an exclusion folder. The folder is excluded only when it
   appears as a direct child of the working directory.

4. **Disabling Ignore File Logic:**
    - **--no-gitignore:** Prevents the tool from reading the **.gitignore** file.
    - **--no-ignore:** Prevents the tool from reading the **.ignore** file.

5. **Command Abbreviations:**  
   Short forms **t** for **tree** and **c** for **content** are now supported.