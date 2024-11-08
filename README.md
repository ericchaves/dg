<p align="center">
  <img src="assets/cover.png" alt="drawing" width="350"/>
</p>

A fast data generator that produces CSV files from generated relational data.

## Table of Contents

1. [Installation](#installation)
1. [Usage](#usage)
   - Import via [HTTP](#import-via-http)
   - Import via [psql](#import-via-psql)
   - Import via [nodelocal](#import-via-nodelocal)
1. [Tables](#tables)
   - [gen](#gen)
   - [const](#const)
   - [set](#set)
   - [inc](#inc)
   - [ref](#ref)
   - [each](#each)
   - [range](#range)
   - [match](#match)
   - [Experimental Features](#experimental-features)
     - [range from features](#range-from-features)
     - [gen templates](#gen-templates)
     - [cuid2](#cuid2)
     - [expr](#expr)
     - [rand](#rand)
     - [rel_date](#rel_date)
     - [case](#case)
     - [fk](#fk)
     - [map](#map)
     - [pick](#pick)
     - [lookup](#lookup)
     - [dist](#dist)
     - [breaking configuration files](#breaking-configuration-files)
1. [Inputs](#inputs)
   - [csv](#csv)
1. [Functions](#functions)
1. [Thanks](#thanks)
1. [Todos](#todos)

### Installation

Find the release that matches your architecture on the [releases](https://github.com/codingconcepts/dg/releases) page.

Download the tar, extract the executable, and move it into your PATH:

```
$ tar -xvf dg_[VERSION]-rc1_macOS.tar.gz
```

### Usage

```
$ dg
Usage dg:
  -c string
        the absolute or relative path to the config file
  -cpuprofile string
        write cpu profile to file
  -i string
        write import statements to file
  -o string
        the absolute or relative path to the output dir (default ".")
  -p int
        port to serve files from (omit to generate without serving)
  -version
        display the current version number
```

Create a config file. In the following example, we create 10,000 people, 50 events, 5 person types, and then populate the many-to-many `person_event` resolver table with 500,000 rows that represent the Cartesian product between the person and event tables:

```yaml
tables:
  - name: person
    count: 10000
    columns:
      # Generate a random UUID for each person
      - name: id
        type: gen
        processor:
          value: ${uuid}

  - name: event
    count: 50
    columns:
      # Generate a random UUID for each event
      - name: id
        type: gen
        processor:
          value: ${uuid}

  - name: person_type
    count: 5
    columns:
      # Generate a random UUID for each person_type
      - name: id
        type: gen
        processor:
          value: ${uuid}

      # Generate a random 16 bit number and left-pad it to 5 digits
      - name: name
        type: gen
        processor:
          value: ${uint16}
          format: "%05d"

  - name: person_event
    columns:
      # Generate a random UUID for each person_event
      - name: id
        type: gen
        processor:
          value: ${uuid}

      # Select a random id from the person_type table
      - name: person_type
        type: ref
        processor:
          table: person_type
          column: id

      # Generate a person_id column for each id in the person table
      - name: person_id
        type: each
        processor:
          table: person
          column: id

      # Generate an event_id column for each id in the event table
      - name: event_id
        type: each
        processor:
          table: event
          column: id
```

Run the application:

```
$ dg -c your_config_file.yaml -o your_output_dir -p 3000
loaded config file                       took: 428µs
generated table: person                  took: 41ms
generated table: event                   took: 159µs
generated table: person_type             took: 42µs
generated table: person_event            took: 1s
generated all tables                     took: 1s
wrote csv: person                        took: 1ms
wrote csv: event                         took: 139µs
wrote csv: person_type                   took: 110µs
wrote csv: person_event                  took: 144ms
wrote all csvs                           took: 145ms
```

This will output and dg will then run an HTTP server allow you to import the files from localhost.

```
your_output_dir
├── event.csv
├── person.csv
├── person_event.csv
└── person_type.csv
```

##### Import via HTTP

Then import the files as you would any other; here's an example insert into CockroachDB:

```sql
IMPORT INTO "person" ("id")
CSV DATA (
    'http://localhost:3000/person.csv'
)
WITH skip='1', nullif = '', allow_quoted_null;

IMPORT INTO "event" ("id")
CSV DATA (
    'http://localhost:3000/event.csv'
)
WITH skip='1', nullif = '', allow_quoted_null;

IMPORT INTO "person_type" ("id", "name")
CSV DATA (
    'http://localhost:3000/person_type.csv'
)
WITH skip='1', nullif = '', allow_quoted_null;

IMPORT INTO "person_event" ("person_id", "event_id", "id", "person_type")
CSV DATA (
    'http://localhost:3000/person_event.csv'
)
WITH skip='1', nullif = '', allow_quoted_null;
```

##### Import via psql

If you're working with a remote database and have access to the `psql` binary, try importing the CSV file as follows:

```sh
psql "postgres://root@localhost:26257/defaultdb?sslmode=disable" \
-c "\COPY public.person (id, full_name, date_of_birth, user_type, favourite_animal) FROM './csvs/person/person.csv' WITH DELIMITER ',' CSV HEADER NULL E''"
```

##### Import via nodelocal

If you're working with a remote database and have access to the `cockroach` binary, try importing the CSV file as follows:

```sh
cockroach nodelocal upload ./csvs/person/person.csv imports/person.csv \
  --url "postgres://root@localhost:26257?sslmode=disable"
```

Then importing the file as follows:

```sql
IMPORT INTO person ("id", "full_name", "date_of_birth", "user_type", "favourite_animal")
  CSV DATA (
    'nodelocal://1/imports/person.csv'
  ) WITH skip = '1';
```

### Tables

Table elements instruct dg to generate data for a single table and output it as a csv file. Here are the configuration options for a table:

```yaml
tables:
  - name: person
    unique_columns: [col_a, col_b]
    count: 10
    columns: ...
```

This config generates 10 random rows for the person table. Here's a breakdown of the fields:

| Field Name     | Optional | Description                                                                                                                  |
| -------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------- |
| name           | No       | Name of the table. Must be unique.                                                                                           |
| unique_columns | Yes      | Removes duplicates from the table based on the column names provided                                                         |
| count          | Yes      | If provided, will determine the number of rows created. If not provided, will be calculated by the current table size.       |
| suppress       | Yes      | If `true` the table won't be written to a CSV. Useful when you need to generate intermediate tables to combine data locally. |
| columns        | No       | A collection of columns to generate for the table.                                                                           |

#### Processors

dg takes its configuration from a config file that is parsed in the form of an object containing arrays of objects; `tables` and `inputs`. Each object in the `tables` array represents a CSV file to be generated for a named table and contains a collection of columns to generate data for.

##### gen

Generate a random value for the column. Here's an example:

```yaml
- name: sku
  type: gen
  processor:
    value: SKU${uint16}
    format: "%05d"
```

This configuration will generate a random left-padded `uint16` with a prefix of "SKU" for a column called "sku". `value` contains zero or more function placeholders that can be used to generate data. A list of available functions can be found [here](https://github.com/codingconcepts/dg#functions).

Generate a pattern-based value for the column. Here's an example:

```yaml
- name: phone
  type: gen
  processor:
    pattern: \d{3}-\d{3}-\d{4}
```

This configuration will generate US-format phone number, like 123-456-7890.

##### const

Provide a constant set of values for a column. Here's an example:

```yaml
- name: options
  type: const
  processor:
    values: [bed_breakfast, bed]
```

This configuration will create a column containing two rows.

##### set

Select a value from a given set. Here's an example:

```yaml
- name: user_type
  type: set
  processor:
    values: [admin, regular, read-only]
```

This configuration will select between the values "admin", "regular", and "read-only"; each with an equal probability of being selected.

Items in a set can also be given a weight, which will affect their likelihood of being selected. Here's an example:

```yaml
- name: favourite_animal
  type: set
  processor:
    values: [rabbit, dog, cat]
    weights: [10, 60, 30]
```

This configuration will select between the values "rabbit", "dog", and "cat"; each with different probabilities of being selected. Rabbits will be selected approximately 10% of the time, dogs 60%, and cats 30%. The total value doesn't have to be 100, however, you can use whichever numbers make most sense to you.

##### inc

Generates an incrementing number. Here's an example:

```yaml
- name: id
  type: inc
  processor:
    start: 1
    format: "P%03d"
```

This configuration will generate left-padded ids starting from 1, and format them with a prefix of "P".

##### ref

References a value from a previously generated table. Here's an example:

```yaml
- name: ptype
  type: ref
  processor:
    table: person_type
    column: id
```

This configuration will choose a random id from the person_type table and create a `ptype` column to store the values.

Use the `ref` type if you need to reference another table but don't need to generate a new row for _every_ instance of the referenced column.

##### each

Creates a row for each value in another table. If multiple `each` columns are provided, a Cartesian product of both columns will be generated.

Here's an example of one `each` column:

```yaml
- name: person
  count: 3
  columns:
    - name: id
      type: gen
      processor:
        value: ${uuid}

# person
#
# id
# c40819f8-2c76-44dd-8c44-5eef6a0f2695
# 58f42be2-6cc9-4a8c-b702-c72ab1decfea
# ccbc2244-667b-4bb5-a5cd-a1e9626a90f9

- name: pet
  columns:
    - name: person_id
      type: each
      processor:
        table: person
        column: id
    - name: name
      type: gen
      processor:
        value: first_name
# pet
#
# person_id                            name
# c40819f8-2c76-44dd-8c44-5eef6a0f2695 Carlo
# 58f42be2-6cc9-4a8c-b702-c72ab1decfea Armando
# ccbc2244-667b-4bb5-a5cd-a1e9626a90f9 Kailey
```

Here's an example of two `each` columns:

```yaml
- name: person
  count: 3
  columns:
    - name: id
      type: gen
      processor:
        value: ${uuid}

# person
#
# id
# c40819f8-2c76-44dd-8c44-5eef6a0f2695
# 58f42be2-6cc9-4a8c-b702-c72ab1decfea
# ccbc2244-667b-4bb5-a5cd-a1e9626a90f9

- name: event
  count: 3
  columns:
    - name: id
      type: gen
      processor:
        value: ${uuid}

# event
#
# id
# 39faeb54-67d1-46db-a38b-825b41bfe919
# 7be981a9-679b-432a-8a0f-4a0267170c68
# 9954f321-8040-4cd7-96e6-248d03ee9266

- name: person_event
  columns:
    - name: person_id
      type: each
      processor:
        table: person
        column: id
    - name: event_id
      type: each
      processor:
        table: event
        column: id
# person_event
#
# person_id
# c40819f8-2c76-44dd-8c44-5eef6a0f2695 39faeb54-67d1-46db-a38b-825b41bfe919
# c40819f8-2c76-44dd-8c44-5eef6a0f2695 7be981a9-679b-432a-8a0f-4a0267170c68
# c40819f8-2c76-44dd-8c44-5eef6a0f2695 9954f321-8040-4cd7-96e6-248d03ee9266
# 58f42be2-6cc9-4a8c-b702-c72ab1decfea 39faeb54-67d1-46db-a38b-825b41bfe919
# 58f42be2-6cc9-4a8c-b702-c72ab1decfea 7be981a9-679b-432a-8a0f-4a0267170c68
# 58f42be2-6cc9-4a8c-b702-c72ab1decfea 9954f321-8040-4cd7-96e6-248d03ee9266
# ccbc2244-667b-4bb5-a5cd-a1e9626a90f9 39faeb54-67d1-46db-a38b-825b41bfe919
# ccbc2244-667b-4bb5-a5cd-a1e9626a90f9 7be981a9-679b-432a-8a0f-4a0267170c68
# ccbc2244-667b-4bb5-a5cd-a1e9626a90f9 9954f321-8040-4cd7-96e6-248d03ee9266
```

Use the `each` type if you need to reference another table and need to generate a new row for _every_ instance of the referenced column.

When a `count` is defined for a table with columns specified as `each`, the Cartesian product of these columns will be iterated over until the specified row `count` is reached.

If the length of the Cartesian product is greater than `count`, not every combination of the specified columns will be used. Conversely, if the length of the Cartesian product is smaller than `count`, some combinations of the specified columns will be duplicated to meet the required row `count`.

##### range

Generates data within a given range. Note that a number of factors determine how this generator will behave. The step (and hence, number of rows) will be generated in the following priority order:

1. If an `each` generator is being used, step will be derived from that
1. If a `count` is provided, step will be derived from that
1. Otherwise, `step` will be used

Here's an example that generates monotonically increasing ids for a table, starting from 1:

```yaml
- name: users
  count: 10000
  columns:
    - name: id
      type: range
      processor:
        type: int
        from: 1
        step: 1
```

Here's an example that generates all dates between `2020-01-01` and `2023-01-01` at daily intervals:

```yaml
- name: event
  columns:
    - name: date
      type: range
      processor:
        type: date
        from: 2020-01-01
        to: 2023-01-01
        step: 24h
        format: 2006-01-02
```

Here's an example that generates 10 dates between `2020-01-01` and `2023-01-02`:

```yaml
- name: event
  count: 10
  columns:
    - name: date
      type: range
      processor:
        type: date
        from: 2020-01-01
        to: 2023-01-01
        format: 2006-01-02
        step: 24h # Ignored due to table count.
```

Here's an example that generates 20 dates (one for every row found from an `each` generator) between `2020-01-01` and `2023-01-02`:

```yaml
- name: person
  count: 20
  columns:
    - name: id
      type: gen
      processor:
        value: ${uuid}

- name: event
  count: 10 # Ignored due to resulting count from "each" generator.
  columns:
    - name: person_id
      type: each
      processor:
        table: person
        column: id

    - name: date
      type: range
      processor:
        type: date
        from: 2020-01-01
        to: 2023-01-01
        format: 2006-01-02
```

The range generate currently supports the following data types:

- `date` - Generate dates between a from and to value
- `int` - Generate integers between a from and to value

There are two additional ways to define the starting value, beyond the `from` attribute. Please check the [range from features](#range-from-features)

##### match

Generates data by matching data in another table. In this example, we'll assume there's a CSV file for the `significant_event` input that generates the following table:

| date       | event |
| ---------- | ----- |
| 2023-01-10 | abc   |
| 2023-01-11 |       |
| 2023-01-12 | def   |

```yaml
inputs:
  - name: significant_event
    type: csv
    source:
      file_name: significant_dates.csv

tables:
  - name: events
    columns:
      - name: timeline_date
        type: range
        processor:
          type: date
          from: 2023-01-09
          to: 2023-01-13
          format: 2006-01-02
          step: 24h
      - name: timeline_event
        type: match
        processor:
          source_table: significant_event
          source_column: date
          source_value: events
          match_column: timeline_date
```

dg will match rows in the significant_event table with rows in the events table based on the match between `significant_event.date` and `events.timeline_date`, and take the value from the `significant_events.event` column where there's a match (otherwise leaving `NULL`). This will result in the following `events` table being generated:

| timeline_date | timeline_event |
| ------------- | -------------- |
| 2023-01-09    |                |
| 2023-01-10    | abc            |
| 2023-01-11    |                |
| 2023-01-12    | def            |
| 2023-01-13    |                |


### Experimental Features

The following features and generators where recently added and may contain bugs.

#### range-from-features

There are two additional ways to define the starting value, beyond the `from` attribute:

1. **Run an external `cmd`**: The generator can execute an external command defined with `cmd` attribute, and it will use the output from stdout as the starting value for from. Its important that the output is either a valid integer or valid string date, compatible with the expected value in `from`.
1. **Pickup up from a source `table`**: You can specify the `table` attribute to have the range generator continue from the last value of the same column in source table. 

These mechanisms allow, for example, the creation of partitioned files where you declare multiple tables named `customers.1`, `customers.2`, and so on. Each table will have the same column definitions, but their generators will have different parameters to cover different scenarios.

The generator does not have access to the source table's `columns` definition, so it's important to ensure that all column definitions are consistent across each generator, according to the specification of the base table.

```yaml
tables:
  - name: events.1
    count: 10
    columns:
      - name: id
        type: range
        processor:
          type: int
          from: 1
          step: 1
  - name: events.2
    columns:
      - name: id
        type: range
        processor:
          table: events.1
          type: int
          step: 1
```

#### gen templates

You canuse [go-fakeit](https://pkg.go.dev/github.com/brianvoe/gofakeit/v7) functions and types with a `template` in a `gen` generator:

```yaml
  - name: rating
    type: gen
    processor:
      template: '{{starrating}}'
  - name: comment
    type: gen
    processor:
      template: '{{setence}}'
  - name: description
    type: gen
    processor:
      template: '{{LoremIpsumSentence 10}}'
```

#### cuid2

Alternatively to UUIDs you cans use [`cuid2`](https://pkg.go.dev/github.com/nrednav/cuid2). For more information about Cuid2 please refer to the [original documentation](https://github.com/paralleldrive/cuid2).

```yaml
  - name: id
    type: cuid2
    processor:
      length: 14
```

#### expr

The `expr` generator enable arithmetic/strings expressions evaluation using [expr-lang](https://expr-lang.org/docs/language-definition).

```yaml
  - name: silly_value
    type: expr
    processor:
      expression: 14 + 33
```

You can use `format` attribute to ensure the requirements of your data shape:

```yaml
- name: formatted_value
    type: expr
    processor:
      expression: 14 / 33
      format: '%.2f'
```

If expression returns an array of values, each value will be used to generate the column value without exceeding the current table count.

```yaml
- name: formatted_value
    type: expr
    processor:
      expression: 1..3
```

Values from the same table row can be used in the expression by using the name of the column.
Since column values are generated as strings, you must use expr's type functions to convert the string value into the the proper type for the operation. 

```yaml
  - name: installments
    type: rand
    processor:
      type: int
      low: 2
      high: 12
  - name: due_amount
    type: rand
    processor:
      type: float64
      low: 1000.0
      high: 2000.0
      format: '%.2f'
  - name: installment_value
    type: expr
    processor:
      expression: 'float(due_amount) / int(installments)'
      format: '%.4f'
```

AML can be tricky when handling quotes, so when an expression requires quotes (such as when returning string values) or becomes more complex, we recommend using block-style indicators (`|` or `>`) for writing expressions.

```yaml
  - name: const_value
    type: expr
    processor:
      expression: >
        'hello!'
  - name: float_value
    type: expr
    processor:
      expression: >
        float(due_amount) / int(installments)
```
You can also reference other tables values using the `match` function in an expression. The `match`function works pretty much like [match](#match) generator, expecting 4 input string parameters: `source_table`,`source_column`,`source_value` and `match_column`.

```yaml
tables:
  - name: persons
    count: 10
    columns:
      - name: person_id
        type: range
        processor:
          type: int
          from: 1
          step: 1
      - name: salary
        type: rand
        processor:
          type: float64
          low: 2000.0
          high: 5000.0
          format: '%.2f'
  - name: loans
    count: 10
    columns:
      - name: loan_id
        type: range
        processor:
          type: int
          from: 1
          step: 1
      - name: max_loan
        type: expr
        processor:
          expression: float(match('persons','person_id', loan_id, 'salary')) / 0.3
          format: '%.2f'
```

The generator adds custom functions to extend those provided by expr-lang, including a wrapper for [gofakeit](https://pkg.go.dev/github.com/brianvoe/gofakeit/v) that allows calling any gofakeit function.

```yaml
tables:
  - name: persons
    count: 10
    columns:
      - name: person_id
        type: range
        processor:
          type: int
          from: 1
          step: 1
      - name: email
        type: gen
        processor:
          template: '{{Email}}'
      - name: salary
        type: rand
        processor:
          type: float64
          low: 2000.0
          high: 5000.0
          format: '%.2f'
      - name: fake_name
        type: expr
        processor:
          expression: fakeit('name', {})
      - name: fake_email
        type: expr
        processor:
          expression: fakeit('email', {})
      - name: fake_phone
        type: expr
        processor:
          expression: fakeit('phone', {})
      - name: fake_sentence
        type: expr
        processor:
          expression: > 
            fakeit('sentence', {'wordCount': 5})
      - name: fake_number
        type: expr
        processor:
          expression: >
            fakeit('number', {'min': 1, 'max': 100})
```

The list of custom functions available are:

- **match(sourceTable string, sourceColumn string, sourceValue string, matchColumn string)** (any, string): *returns the `matchColumn` from `sourceTable` where `sourceColumn` has `SourceValue`*
- **add_date(years int, months int, days int, date any)** (any, error): *Adds the specified numbers of `years`,`months` and `days` to the given `date`*
- **rand(n int)** int: *returns a pseudo-random between 0 and `n` when n is positive and between -n and o when n is negative*.
- **randr(min int, max int)** int: *returns a pseudo-random integer between `min` and `max`, inclusive. Accepts both positive and negative values*
- **get_record(table string, line int)** (map[string]any, error): *returns a map[string]any with the row value for a given line of a in memory (processed) table*
- **get_column(table string, column string)** ([]string, error): *returns a [string]string with all column values of a in memory (processed) table*
- **get_model(table string)** (CSVFile, error): *returns the in memory CSVFile struct of the given table*
- **payments(total float64, installments int, percentage float64)** ([]float64, error) *returns a []float64, calculates the down payment and equal installment amounts based on a specified down payment percentage and number of installments*
- **pmt(rate float64, nper int, pv float64, fv float64, type int)** (float64, error): *returns float64 fixed payment (principal + rate of interest) against a loan (fv=0) or future value given a initial deposit (pv). type indicates whether payment is made at the beginning (1) or end (0) of each period (nper)*
- **fakeit(name string, params map[string]any)** (any, error): *call a function from [gofakeit](https://pkg.go.dev/github.com/brianvoe/gofakeit/v7) by passing a map with the required arguments. the return value depends on the specific function called.*
- **sha256(data s)** string: *returns the SHA256 checksum of the data.*
- **pad(s string, char string, length int, left bool)** string: *returns the string padded with char to the left or right.*
- **slug(s string, lang string)** string: *returns the slugfied string. if lang is blank "en" is assumed.*

#### rand

`rand` generator allows generation of random values between a given range providing a `low`and `high` values (both inclusive). Supported types are `int`, `date` and `float64`. 

```yaml
  - name: age
    type: rand
    processor:
      type: int
      low: 10
      high: 20
  - name: enrollment_date
    type: rand
    processor:
      type: date
      low: '2010-01-01'
      high: '2020-01-01'
  - name: salary
    type: rand
    processor:
      type: float64
      low: 1000.0
      high: 2000.0
      format: '%.2f'
```

You can adjust output values providing a `format` parameter.

For `date` types the `format`, when provided, is also used to parse the date values provided in `low` and `high` parameters, otherwise `'2006-01-02'` is used as default.

For detailed information on date layouts (formats) check out [go/time documention](https://pkg.go.dev/time#pkg-constants).

#### rel_date

The `rel_date` generator allows for the generation of random dates relative to a given reference date. For example, using the `after` and `before` attributes, you can set dates within a range, such as from 7 days before to 5 days after the current date (values are inclusive).

The `unit` specifies the time span unit. Allowed values are `day`, `month`, and `year`.

You can provide a date layout using the [Go time documentation](https://pkg.go.dev/time#pkg-constants) to `format` the output value.

The `date` parameter is optional and if not provided the current date (`'now'`) is assumed. When format is specified, the `date` must be in the same layout. 

You can pass a expr expression on `date`,`before` and `after` parameters, which allows reference other values in the same row by providing the column name , use the match function to match other tables values or perform some complex calculations. `before` and `after` expressions evaluate to `int` while `date` must evaluate to a `date` type.

```yaml
  - name: relative_from_now
    type: rel_date
    processor:
      unit: day
      after: '-7'
      before: '7'
      format: '02/01/2006'
  - name: relative_from_date
    type: rel_date
    processor:
      date: '2020-12-25'
      unit: year
      after: '-4'
      before:'4'
      format: '2006-01-02'
  - name: relative_from_other_column
    type: rel_date
    processor:
      date: 'other_column_name'
      unit: year
      after: '-4'
      before: '4'
      format: '2006-01-02'
  - name: before_after_from_other_column
    type: rel_date
    processor:
      date: 'now'
      unit: year
      after: 'int(some_column)'
      before: 'int(another_column)'
      format: '2006-01-02'
```
#### case

The `case` generator evaluates a set of conditions composed of `when` and `then` expressions.
Each condition is evaluated in order, and when the first `when` expression evaluates to `true` the `then` is evaluated to produce the column value.
All functionalities of the [expr generator](#expr) can be used in the `When` and `Value` expressions. 

YAML can be tricky when handling quotes, so we recommend using block-style indicators (`|` or `>`) for writing expressions.

```yaml
- name: greeting
  type: case
  processor:
    - when: > 
        int(age) <= 5 && gender == 'male'
      then: > 
        'Little guy'
    - when: > 
        int(age) <= 5 && gender == 'female'
      then: > 
        'Little girl'
    - when: > 
      'true'
      then: > 
        'Hi'
```

Each condition can define it's own `format` attribute to control the formatted output value.

```yaml
- name: revenue
  type: case
  processor:
    - when: >
        int(age) <= 5
      then: >
        float(allowance)
      format: '%.2f'
    - when: > 
        int(age) > 5
      then: >
        float(salary)
      format: '%.3f'
```

#### FK

The `fk` generator is usefull when we need to reference value from a parent `table` to build 1:1 or 1:N relations.

The `repeat` parameter is an [expr](#expr) expression that must evaluate to an integer, which will determine how many times each parent value should be used. 
You can reference parent columns in the `repeat` expression using the notation `parent.column_name`.

You can apply a `filter` [expr](#expr) expression that evaluates to a boolean, allowing you to skip certain rows. When a filter is applied, the number of `skipped` rows is stored in a custom integer variable, which can be used in both `repeat` and `filter` expressions. 

When the table being generated includes a `count` value, it will restrict the number of values created to match the `count`, even if the `repeat` value is larger than the `count`.

```yaml
tables:
  - name: orders
    count: 5
    columns:    
      - name: order_id
        type: range
        processor:
          type: int
          from: 1
          step: 1    
      - name: item_count
        type: rand
        processor:
          type: int
          low: 1
          high: 10
  - name: order_items
    columns:
      - name: id
        type: range
        processor:
          type: int
          from: 1
          step: 1
      - name: order_id
        type: fk 
        processor:
          table: orders 
          column: order_id
          repeat: 'int(parent.item_count)'
      - name: item_name
        type: gen
        processor:
          value: ${breakfast}
```

#### map

The `map` generator maps each value in a specified column from a source table and generates a new column using the `expression`. This generator is useful for creating distributions or frequency-based data sets.

The following custom variables are available for use in expressions:

* **row_number**: Represents the generated line number, provided as an integer. Start with 0.
* **value**: Refers to the current value, formatted as a string.
* **count**: Indicates the number of occurrences of **value**, expressed as an integer.
* **index**: Denotes the current iterator number for **value** (e.g., 1 of 3, 2 of 3, etc.) starting with 1.

**Example** YAML configuration:

```yaml
- name: value_counts
  type: map
  processor:
    table: source_table
    column: category
    expression: "string(index) + ':' + string(value) + ':' + string(count)"
```
If you want to count values from another column of the current table you can ommit `table` parameter.
You can use `format` to control the formatted output value. 

### pick

The `pick` generator retrieves values from `source_value` by matching the `match_column` in the current table with the `source_column` in the `source_table`.

The `Unique` parameter is optional; when set to `true`, the generator ensures each value is used only once. If `Unique` is `false` (the default), the generator will cycle through the available values in the source, reusing them as needed to populate the target table

**Example** YAML configuration:

```yaml
- name: unique_assignment
  type: link
  processor:
    source_table: source_table
    source_column: unique_ids
    source_value: value_column
    match_column: match_column
    unique: true
```

In this example, the generator will:
1. Look up values in the 'source_table'.
2. Match the 'match_column' in the current table with the 'source_column' in the source table.
3. When a match is found, it will assign the corresponding value from the 'source_value' column.
4. Each value from 'source_value' is used only once.

#### lookup

The `lookup` generator allows you to generate values for columns in a table based on search logic across multiple tables.

It behaves like a join, starting from the current table as the base table and searching for each value in the `match_column`. The generator will pick the first item from the `tables` list and look for the current value in `source_table`.`source_column`. Once a match is found, either the value from `source_table`.`source_value` column is returned, or the output of the `expression`. This value will become the search value for the next item in `tables` until it reaches the end of the list, returning the last value to be used by the generator as the new column value.

The `lookup` generator does not produce cartesian products; when it finds a value in the searched table, it stops the search and moves to the next table. In cases where the searched table has more than one value, we can use the `predicate` filter to instruct whether the current match should be used or if it should be skipped and the search should continue for another match.

The `ignore_missing` attribute determines how the generator handles lookup failures. When `true`, the generator will ignore missing values and continue processing, using an empty value when no match is found, similar to a left join. When `false`, it stops and returns an error when a value is not found. When omitted the value `false` is assumed (default). 

The following custom variables are available for use in expressions:

* **row_number**: Represents the line number where the match happened, provided as an integer, starting from 0.
* **rows_skipped**: Represents the number of matches skipped until the current match is returned, provided as an integer, starting from 0.

Those values are reset for each table search.

```yaml
- name: last_purchase
  type: lookup
  processor:
    ignore_missing: true
    match_column: "customer_name"
    tables:
    - source_table: "customers"  # Name of the lookup table
      source_column: "name"      # Column in the lookup table to match
      source_value: "id"         # Column whose value should be returned
    - source_table: "orders"
      source_column: "customer_id"
      source_value: "order_date"
```

In this example, the generator will:
1. Start by finding the `customer_name` in the `customers` table.
1. Then, it retrieves the value under the `id` column for the same row.
1. Next, the returned `id` is used to search in the `orders` table under the `customer_id` column.
1. If a match is found, it retrieves the value from the `order_date` column.
1. The retrieved `order_date` value is used to generate new data for the target column in the base table.

Each lookup in table can declare an [expr](#expr) `expression` to return a calculated value when the match is found.
Additionally, a `predicate` expression may be declared which will be evaluated when a match is found. If the predicate evaluates to `false`, the value is skipped, and the search continues, looking for another match, allowing different results to be returned using the `expression`.

You can set `format` for each table to control the formatted output value.

```yaml
- name: last_purchase
  type: lookup
  processor:
    ignore_missing: true
    match_column: "customer_name"
    tables:
    - source_table: "customers"  # Name of the lookup table
      source_column: "name"      # Column in the lookup table to match
      source_value: "id"         # Column whose value should be returned
    - source_table: "orders"
      source_column: "customer_id"
      source_value: "order_date"
      predicate: int(order_items) > 5
      format: '02-01-2006'
```

#### dist

The `dist` generator is useful when you need to generate values based on weighted probabilities. This is particularly helpful for scenarios where certain values should appear more frequently than others according to predefined weights.

The `values` parameter contains a list of possible values to be generated, while the `weights` parameter assigns a corresponding weight (integer) to each value. 
The weights determine the likelihood of each value being selected, with higher weights resulting in more frequent occurrences of that value.

For example, the configuration below would result in a column with 70 "dog", 20 "cats" and 10 "birds".

```yaml
tables:
- name: people
  count: 100
  columns:
    - name: pets
      type: dist
      processor:
        values: [ dogs, cats, birds]
        weigths: [ 7, 2, 1]
```
If the weights don't perfectly fill the count, additional values will be added until the desired total is reached. Once generated, the values are shuffled randomly to avoid any predictable order.

`dist` generator accepts an [expr](#expr) `expression` that returns an array of strings whose distinct values will be merged with the `values` array and the count of each value will be added to the `weigths`.

```yaml
tables:
- name: people
  count: 100
  columns:
    - name: pets
      type: dist
      processor:
        expression: |
          [ "dogs", "cats", "birds", "dogs", "cats"]
```
        
The difference between `dist` and [set](#set) lies in how the distribuition is handled. In `set` the values are **randomly selected**, using the weights as probabilities for each selection. In contrast, the `dist` generator ensures that the values are **distributed proportionally** to their weights. In other words, with [set](#set), the outcome can deviate significantly from the weights, while with `dist`, the result will closely match the specified proportions.

#### Breaking configuration files

In complex databases where there is a large number of tables with multiple cardinalities and dependencies, the config file may become too big and hard to maintain, particularly if the database is in a stage where changes are frequent. There are two ways to break down your configuration into multiple files:

1. Using the `extends` section in your YAML files
2. Using multiple `-c` flags when running `dg`

##### Using extends

The `extends` section allows a configuration file to inherit and override settings from other files. For example:

```yaml
# base.yaml
tables:
  - name: users
    columns:
      - name: id
        type: inc

# extra.yaml
extends:
  - base.yaml
tables:
  - name: users
    count: 1000  # Override just the count
```

When using `extends`:
- Files are processed recursively in the order they appear
- Paths in `extends` are relative to the current file's location
- Later configurations override earlier ones using the same merge rules as multiple configs

##### Using Multiple Config Files

You can also pass multiple config files by repeating the config flag:

```bash
dg -c ./path/to/base.yaml -c ./path/to/extra-tables.yaml -c ./path/to/overrides.yaml -o ./output/
```

The merge logic follows these rules when combining configurations:

- Files are merged in pairs from left to right
- For input items:
  - Matching names: incoming item completely replaces base item
  - New items are added
- For tables:
  - Matching names:
    - Count: incoming overrides base (default 0)
    - Suppress flag: incoming overrides base (default false) 
    - Columns: either kept as-is or completely replaced
  - New tables are added

These rules make it easier to:
- Add/replace inputs and tables (but not remove them)
- Override counts and suppress flags without redefining columns
- Split related tables into separate files
- Have override files that just modify counts/flags

For example:
```yaml
# base.yaml - Core tables
tables:
  - name: users
    columns: [...]

# extra.yaml - Additional tables
tables: 
  - name: orders
    columns: [...]

# override.yaml - Just modify counts
tables:
  - name: users
    count: 1000
  - name: orders 
    count: 5000
```

You can use both approaches together - files specified with `-c` can contain `extends` sections, giving you flexible ways to organize your configurations.

### Inputs

dg takes its configuration from a config file that is parsed in the form of an object containing arrays of objects; `tables` and `inputs`. Each object in the `inputs` array represents a data source from which a table can be created. Tables created via inputs will not result in output CSVs.

##### csv

Reads in a CSV file as a table that can be referenced from other tables. Here's an example:

```yaml
- name: significant_event
  type: csv
  source:
    file_name: significant_dates.csv
```

This configuration will read from a file called significant_dates.csv and create a table from its contents. Note that the `file_name` should be relative to the config directory, so if your CSV file is in the same directory as your config file, just include the file name.

### Functions

| Name                           | Type      | Example                                                                                                   |
| ------------------------------ | --------- | --------------------------------------------------------------------------------------------------------- |
| ${ach_account}                 | string    | 586981797546                                                                                              |
| ${ach_routing}                 | string    | 441478502                                                                                                 |
| ${adjective_demonstrative}     | string    | there                                                                                                     |
| ${adjective_descriptive}       | string    | eager                                                                                                     |
| ${adjective_indefinite}        | string    | several                                                                                                   |
| ${adjective_interrogative}     | string    | whose                                                                                                     |
| ${adjective_possessive}        | string    | her                                                                                                       |
| ${adjective_proper}            | string    | Iraqi                                                                                                     |
| ${adjective_quantitative}      | string    | sufficient                                                                                                |
| ${adjective}                   | string    | double                                                                                                    |
| ${adverb_degree}               | string    | far                                                                                                       |
| ${adverb_frequency_definite}   | string    | daily                                                                                                     |
| ${adverb_frequency_indefinite} | string    | always                                                                                                    |
| ${adverb_manner}               | string    | unexpectedly                                                                                              |
| ${adverb_place}                | string    | here                                                                                                      |
| ${adverb_time_definite}        | string    | yesterday                                                                                                 |
| ${adverb_time_indefinite}      | string    | just                                                                                                      |
| ${adverb}                      | string    | far                                                                                                       |
| ${animal_type}                 | string    | mammals                                                                                                   |
| ${animal}                      | string    | ape                                                                                                       |
| ${app_author}                  | string    | RedLaser                                                                                                  |
| ${app_name}                    | string    | SlateBlueweek                                                                                             |
| ${app_version}                 | string    | 3.2.10                                                                                                    |
| ${bitcoin_address}             | string    | 16YmZ5ol5aXKjilZT2c2nIeHpbq                                                                               |
| ${bitcoin_private_key}         | string    | 5JzwyfrpHRoiA59Y1Pd9yLq52cQrAXxSNK4QrGrRUxkak5Howhe                                                       |
| ${bool}                        | bool      | true                                                                                                      |
| ${breakfast}                   | string    | Awesome orange chocolate muffins                                                                          |
| ${bs}                          | string    | leading-edge                                                                                              |
| ${car_fuel_type}               | string    | LPG                                                                                                       |
| ${car_maker}                   | string    | Seat                                                                                                      |
| ${car_model}                   | string    | Camry Solara Convertible                                                                                  |
| ${car_transmission_type}       | string    | Manual                                                                                                    |
| ${car_type}                    | string    | Passenger car mini                                                                                        |
| ${chrome_user_agent}           | string    | Mozilla/5.0 (X11; Linux i686) AppleWebKit/5310 (KHTML, like Gecko) Chrome/37.0.882.0 Mobile Safari/5310   |
| ${city}                        | string    | Memphis                                                                                                   |
| ${cnpj}                        | string    | 63776262000162                                                                                            |
| ${color}                       | string    | DarkBlue                                                                                                  |
| ${company_suffix}              | string    | LLC                                                                                                       |
| ${company}                     | string    | PlanetEcosystems                                                                                          |
| ${connective_casual}           | string    | an effect of                                                                                              |
| ${connective_complaint}        | string    | i.e.                                                                                                      |
| ${connective_examplify}        | string    | for example                                                                                               |
| ${connective_listing}          | string    | next                                                                                                      |
| ${connective_time}             | string    | soon                                                                                                      |
| ${connective}                  | string    | for instance                                                                                              |
| ${country_abr}                 | string    | VU                                                                                                        |
| ${country}                     | string    | Eswatini                                                                                                  |
| ${cpf}                         | string    | 56061433301                                                                                               |
| ${credit_card_cvv}             | string    | 315                                                                                                       |
| ${credit_card_exp}             | string    | 06/28                                                                                                     |
| ${credit_card_type}            | string    | Mastercard                                                                                                |
| ${currency_long}               | string    | Mozambique Metical                                                                                        |
| ${currency_short}              | string    | SCR                                                                                                       |
| ${date}                        | time.Time | 2005-01-25 22:17:55.371781952 +0000 UTC                                                                   |
| ${day}                         | int       | 27                                                                                                        |
| ${dessert}                     | string    | Chocolate coconut dream bars                                                                              |
| ${dinner}                      | string    | Creole potato salad                                                                                       |
| ${domain_name}                 | string    | centralb2c.net                                                                                            |
| ${domain_suffix}               | string    | com                                                                                                       |
| ${email}                       | string    | ethanlebsack@lynch.name                                                                                   |
| ${emoji}                       | string    | ♻️                                                                                                         |
| ${file_extension}              | string    | csv                                                                                                       |
| ${file_mime_type}              | string    | image/vasa                                                                                                |
| ${firefox_user_agent}          | string    | Mozilla/5.0 (X11; Linux x86_64; rv:6.0) Gecko/1951-07-21 Firefox/37.0                                     |
| ${first_name}                  | string    | Kailee                                                                                                    |
| ${flipacoin}                   | string    | Tails                                                                                                     |
| ${float32}                     | float32   | 2.7906555e+38                                                                                             |
| ${float64}                     | float64   | 4.314310154193861e+307                                                                                    |
| ${fruit}                       | string    | Eggplant                                                                                                  |
| ${gender}                      | string    | female                                                                                                    |
| ${hexcolor}                    | string    | #6daf06                                                                                                   |
| ${hobby}                       | string    | Bowling                                                                                                   |
| ${hour}                        | int       | 18                                                                                                        |
| ${http_method}                 | string    | DELETE                                                                                                    |
| ${http_status_code_simple}     | int       | 404                                                                                                       |
| ${http_status_code}            | int       | 503                                                                                                       |
| ${http_version}                | string    | HTTP/1.1                                                                                                  |
| ${int16}                       | int16     | 18940                                                                                                     |
| ${int32}                       | int32     | 2129368442                                                                                                |
| ${int64}                       | int64     | 5051946056392951363                                                                                       |
| ${int8}                        | int8      | 110                                                                                                       |
| ${ipv4_address}                | string    | 191.131.155.85                                                                                            |
| ${ipv6_address}                | string    | 1642:94b:52d8:3a4e:38bc:4d87:846e:9c83                                                                    |
| ${job_descriptor}              | string    | Senior                                                                                                    |
| ${job_level}                   | string    | Identity                                                                                                  |
| ${job_title}                   | string    | Executive                                                                                                 |
| ${language_abbreviation}       | string    | kn                                                                                                        |
| ${language}                    | string    | Bengali                                                                                                   |
| ${last_name}                   | string    | Friesen                                                                                                   |
| ${latitude}                    | float64   | 45.919913                                                                                                 |
| ${longitude}                   | float64   | -110.313125                                                                                               |
| ${lunch}                       | string    | Sweet and sour pork balls                                                                                 |
| ${mac_address}                 | string    | bd:e8:ce:66:da:5b                                                                                         |
| ${minute}                      | int       | 23                                                                                                        |
| ${month_string}                | string    | April                                                                                                     |
| ${month}                       | int       | 10                                                                                                        |
| ${name_prefix}                 | string    | Ms.                                                                                                       |
| ${name_suffix}                 | string    | I                                                                                                         |
| ${name}                        | string    | Paxton Schumm                                                                                             |
| ${nanosecond}                  | int       | 349669923                                                                                                 |
| ${nicecolors}                  | []string  | [#490a3d #bd1550 #e97f02 #f8ca00 #8a9b0f]                                                                 |
| ${noun_abstract}               | string    | timing                                                                                                    |
| ${noun_collective_animal}      | string    | brace                                                                                                     |
| ${noun_collective_people}      | string    | mob                                                                                                       |
| ${noun_collective_thing}       | string    | orchard                                                                                                   |
| ${noun_common}                 | string    | problem                                                                                                   |
| ${noun_concrete}               | string    | town                                                                                                      |
| ${noun_countable}              | string    | cat                                                                                                       |
| ${noun_uncountable}            | string    | wisdom                                                                                                    |
| ${noun}                        | string    | case                                                                                                      |
| ${opera_user_agent}            | string    | Opera/10.10 (Windows NT 5.01; en-US) Presto/2.11.165 Version/13.00                                        |
| ${password}                    | string    | 1k0vWN 9Z                                                                                                 | 4f={B YPRda4ys. |
| ${pet_name}                    | string    | Bernadette                                                                                                |
| ${phone_formatted}             | string    | (476)455-2253                                                                                             |
| ${phone}                       | string    | 2692528685                                                                                                |
| ${phrase}                      | string    | I'm straight                                                                                              |
| ${preposition_compound}        | string    | ahead of                                                                                                  |
| ${preposition_double}          | string    | next to                                                                                                   |
| ${preposition_simple}          | string    | at                                                                                                        |
| ${preposition}                 | string    | outside of                                                                                                |
| ${programming_language}        | string    | PL/SQL                                                                                                    |
| ${pronoun_demonstrative}       | string    | those                                                                                                     |
| ${pronoun_interrogative}       | string    | whom                                                                                                      |
| ${pronoun_object}              | string    | us                                                                                                        |
| ${pronoun_personal}            | string    | I                                                                                                         |
| ${pronoun_possessive}          | string    | mine                                                                                                      |
| ${pronoun_reflective}          | string    | yourself                                                                                                  |
| ${pronoun_relative}            | string    | whom                                                                                                      |
| ${pronoun}                     | string    | those                                                                                                     |
| ${quote}                       | string    | "Raw denim tilde cronut mlkshk photo booth kickstarter." - Gunnar Rice                                    |
| ${rgbcolor}                    | []int     | [152 74 172]                                                                                              |
| ${safari_user_agent}           | string    | Mozilla/5.0 (Windows; U; Windows 95) AppleWebKit/536.41.5 (KHTML, like Gecko) Version/5.2 Safari/536.41.5 |
| ${safecolor}                   | string    | gray                                                                                                      |
| ${second}                      | int       | 58                                                                                                        |
| ${snack}                       | string    | Crispy fried chicken spring rolls                                                                         |
| ${ssn}                         | string    | 783135577                                                                                                 |
| ${state_abr}                   | string    | AL                                                                                                        |
| ${state}                       | string    | Kentucky                                                                                                  |
| ${street_name}                 | string    | Way                                                                                                       |
| ${street_number}               | string    | 6234                                                                                                      |
| ${street_prefix}               | string    | Port                                                                                                      |
| ${street_suffix}               | string    | stad                                                                                                      |
| ${street}                      | string    | 11083 Lake Fall mouth                                                                                     |
| ${time_zone_abv}               | string    | ADT                                                                                                       |
| ${time_zone_full}              | string    | (UTC-02:00) Coordinated Universal Time-02                                                                 |
| ${time_zone_offset}            | float32   | 3                                                                                                         |
| ${time_zone_region}            | string    | Asia/Aqtau                                                                                                |
| ${time_zone}                   | string    | Mountain Standard Time (Mexico)                                                                           |
| ${uint128_hex}                 | string    | 0xcd50930d5bc0f2e8fa36205e3d7bd7b2                                                                        |
| ${uint16_hex}                  | string    | 0x7c80                                                                                                    |
| ${uint16}                      | uint16    | 25076                                                                                                     |
| ${uint256_hex}                 | string    | 0x61334b8c51fa841bf9a3f1f0ac3750cd1b51ca2046b0fb75627ac73001f0c5aa                                        |
| ${uint32_hex}                  | string    | 0xfe208664                                                                                                |
| ${uint32}                      | uint32    | 783098878                                                                                                 |
| ${uint64_hex}                  | string    | 0xc8b91dc44e631956                                                                                        |
| ${uint64}                      | uint64    | 5722659847801560283                                                                                       |
| ${uint8_hex}                   | string    | 0x65                                                                                                      |
| ${uint8}                       | uint8     | 192                                                                                                       |
| ${url}                         | string    | https://www.leadcutting-edge.net/productize                                                               |
| ${user_agent}                  | string    | Opera/10.64 (Windows NT 5.2; en-US) Presto/2.13.295 Version/10.00                                         |
| ${username}                    | string    | Gutmann2845                                                                                               |
| ${uuid}                        | string    | e6e34ff4-1def-41e5-9afb-f697a51c0359                                                                      |
| ${vegetable}                   | string    | Tomato                                                                                                    |
| ${verb_action}                 | string    | knit                                                                                                      |
| ${verb_helping}                | string    | did                                                                                                       |
| ${verb_linking}                | string    | has                                                                                                       |
| ${verb}                        | string    | be                                                                                                        |
| ${weekday}                     | string    | Tuesday                                                                                                   |
| ${word}                        | string    | month                                                                                                     |
| ${year}                        | int       | 1962                                                                                                      |
| ${zip}                         | string    | 45618                                                                                                     |

### Building releases locally

```
$ VERSION=0.1.0 make release
```

### Thanks

Thanks to the maintainers of the following fantastic packages, whose code this tools makes use of:

- [samber/lo](https://github.com/samber/lo)
- [brianvoe/gofakeit](https://github.com/brianvoe/gofakeit)
- [go-yaml/yaml](https://github.com/go-yaml/yaml)
- [stretchr/testify](https://github.com/stretchr/testify/assert)
- [martinusso/go-docs](https://github.com/martinusso/go-docs)
- [expr-lang](https://expr-lang.org/)
- [nrednav/cuid2 ](https://github.com/nrednav/cuid2)
- [gosimple/slug](https://github.com/gosimple/slug)

### Todos

- Improve code coverage
- Write file after generating, then only keep columns that other tables need
- Support for range without a table count (e.g. the following results in zero rows unless a count is provided)

```yaml
- name: bet_types
  count: 3
  columns:
    - name: id
      type: range
      processor:
        type: int
        from: 1
        step: 1
    - name: description
      type: const
      processor:
        values: [Win, Lose, Draw]
```
