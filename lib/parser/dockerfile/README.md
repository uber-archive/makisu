For the most part, this parser is compatible with the official Docker parser. However, there
are a few unintuitive behaviors present in the Docker parser that have been replaced here.

# Variable substitution

All supported directives allow variable substitution from both ARG and ENV directives.

Valid variable names consist of {letters, digits, '-', '\_', '.'}, and variable values can
contain any character.

Variable substitutions can be specified in the following formats:
- $\<var\>
    - Terminates once an invalid variable name character is encountered (e.g., if var=val1 and var\_=val2, /$var/ -> /val1/ and \_$var\_ -> \_$val2).
- ${\<var\>}
    - Supports recursive variable resolution (e.g., if var=val1 and val1=val2, ${$var} -> val2).
- ${\<var\>:+\<default\_val\>}
    - If \<var\> not set, resolves to \<default\_val\>, else the value for \<var\>. \<var\> may contain variables to resolve, but \<default\_val\> may not.
- ${\<var\>:-\<default\_val\>}
    - If \<var\> set, resolves to \<default\_val\>, else the empty string. \<var\> may contain variables to resolve, but \<default\_val\> may not.

If a variable fails to resolve, it is passed through to the resulting string exactly as it appears in the input.

# Directives

The following directives are not supported: ONBUILD and SHELL.

## COMMIT

Syntax:
- #!COMMIT
    - 'COMMIT' can be any case and there can be whitespace preceding '#', after '!', or after 'COMMIT'.
    - It cannot be at the beginning of a line, since all lines beginning with '#' would be ignored. 

This is a special directive that indicates that a layer should be committed (used in the distributed cache). To enable this directive, `--commit=explicit` argument is required.

## ADD

Syntax:
- ADD \[--chown=\<user\>:\<group\>\] \<src\> ... \<dest\>
    - Arguments must be separated by whitespace.
- ADD \[--chown=\<user\>:\<group\>\] \["\<src\>",... "\<dest\>"\] (this form is required for paths containing whitespace)
    - JSON format.

Variables are substituted using values from ARGs and ENVs within the stage.

## CMD

Syntax:
- CMD ["\<arg\>", "\<arg\>"...]
    - JSON format.
- CMD \<cmd\> [\<arg\> ...]
    - \<cmd\> and \<arg\>s must be separated by whitespace.
    - To include whitespace within an argument, the whitespace must be escaped using a backslash character or the argument must be surrounded in quotes.
    - Quotes to be included in an argument must be escaped with a backslash.
    - Any backslash characters present in an argument that don't precede whitespace or a quote will be passed through to the resulting string.

Variables are substituted using values from ARGs and ENVs within the stage.

## COPY

Syntax:
- COPY \[--chown=\<user\>:\<group\>\] \[--from=\<name|index\>\] \<src\> ... \<dest\>
    - Arguments must be separated by whitespace.
- COPY \[--chown=\<user\>:\<group\>\] \[--from=\<name|index\>\] \["\<src\>",... "\<dest\>"\] (this form is required for paths containing whitespace)
    - JSON format.

Variables are substituted using values from ARGs and ENVs within the stage.

## ENTRYPOINT

Syntax:
- ENTRYPOINT ["\<arg\>", "\<arg\>"...]
    - JSON format.
- ENTRYPOINT \<cmd\> [\<arg\> ...]
    - \<cmd\> and \<arg\>s must be separated by whitespace. To include whitespace within a single argument, the whitespace must be escaped using a backslash character or the argument must be surrounded in quotes. Quotes within an argument must also be escaped with a backslash. Any backslash characters present in an argument that don't precede whitespace or a quote will be passed through to the resulting string.

Variables are substituted using values from ARGs and ENVs within the stage.

## ENV

Syntax:
- ENV \<key\> \<value\>
    - Everything after the first space character after \<key\> is included in \<value\>.
- ENV \<key\>=\<value\> ...
    - \<key\>=\<value\> pairs must be separated by whitespace.
    - Valid \<key\> characters are: letters, digits, '-', '\_', and '.'.
    - \<value\>s may contain any character, but to include whitespace it must be escaped using a backslash character or the argument must be surrounded in quotes.
    - Quotes to be included in a \<value\> must be escaped with a backslash.

## EXPOSE

Syntax:
- EXPOSE \<port\>[/\<protocol\>] ...
    - Arguments must be separated by whitespace.

Variables are substituted using values from ARGs and ENVs within the stage.

## FROM

Syntax:
- FROM \<image\> [AS \<name\>]

Variables are substituted using globally defined ARGs (those that appear before the first FROM directive).

## HEALTHCHECK

Syntax:
- HEALTHCHECK NONE
- HEALTHCHECK \[--interval=\<time\>\] \[--timeout=\<time\>\] \[--start-period=\<time\>\] \[--retries=\<n\>\] CMD ["\<arg\>", "\<arg\>"...]
    - CMD section is in JSON format.
- HEALTHCHECK \[--interval=\<time\>\] \[--timeout=\<time\>\] \[--start-period=\<time\>\] \[--retries=\<n\>\] CMD \<full\_cmd\>
    - \<full\_cmd\> will be added to healthcheck section of image config as-is (after variable substitution).

Variables after CMD are substituted using values from ARGs and ENVs within the stage.

## LABEL

Syntax:
- LABEL \<key\>=\<value\> ...
    - \<key\>=\<value\> pairs must be separated by whitespace.
    - Valid \<key\> characters are: letters, digits, '-', '\_', and '.'.
    - \<value\>s may contain any character, but to include whitespace it must be escaped using a backslash character or the argument must be surrounded in quotes.
    - Quotes to be included in a \<value\> must be escaped with a backslash.

Variables are substituted using values from ARGs and ENVs within the stage.

## MAINTAINER

Syntax:
- MAINTAINER \<maintainer\>

Variables are not substituted.

## RUN

Syntax:
- RUN ["\<arg\>", "\<arg\>"...]
    - JSON format.
- RUN \<full\_cmd\>
    - \<full\_cmd\> will be passed to shell via 'sh -c' as-is (after variable substitution).

Variables are substituted using values from ARGs and ENVs within the stage.

## STOPSIGNAL

Syntax:
- STOPSIGNAL \<signal\>

Variables are not substituted.

## USER

Syntax:
- USER \<user\>:[\<group\>]
    - Can be specified by user/group name or user/group ID.

Variables are substituted using values from ARGs and ENVs within the stage.

## VOLUME

Syntax:
- VOLUME ["\<volume\>", "\<volume\>"...]
- VOLUME \<volume\> [\<volume\> ...]
    - Volumes must be separated by whitespace characters.

Variables are substituted using values from ARGs and ENVs within the stage.

## WORKDIR

Syntax:
- WORKDIR \<path\>

Variables are substituted using values from ARGs and ENVs within the stage.

## ARG

Syntax:
- ARG \<name\>[=\<default\_val\>]

If after the first FROM directive, variables are substituted into the directive using values from ARGs and ENVs within the stage. Else, variables are only substituted using values from other ARG directives that appeared prior to this one.

Variables defined by ARG directives before the first FROM are used only by all FROM directives. Those defined within a stage are scoped to that stage only.
